package builder

import (
	"fmt"
	"net/url"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func buildQuery(msg proto.Message, usedFieldsPath *usedFields, usedFieldBody string) (url.Values, error) {
	var (
		query      = url.Values{}
		msgReflect = msg.ProtoReflect()
	)

	fields := msgReflect.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		var (
			fd        = fields.Get(i)
			fieldName = fd.JSONName()
		)

		if usedFieldsPath.hasFullKey(fieldName) {
			continue
		}
		if fieldName == usedFieldBody {
			continue
		}

		val := msgReflect.Get(fd)
		if isZeroValue(val, fd) {
			continue
		}

		// Note: order of the cases is important!
		switch {
		case fd.IsList():
			if fd.Kind() == protoreflect.MessageKind {
				return nil, fmt.Errorf("repeated message field %s cannot be mapped to URL query parameters", fieldName)
			}
			list := val.List()
			for j := 0; j < list.Len(); j++ {
				strVal, err := stringifyValue(list.Get(j), fd)
				if err != nil {
					return nil, fmt.Errorf("stringify value for query %s: %w", fieldName, err)
				}
				query.Add(fieldName, strVal)
			}

		case fd.IsMap():
			var (
				m        = val.Map()
				rangeErr error
			)
			m.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
				key := fmt.Sprintf("%s.%s", fieldName, k.String())

				if fd.MapValue().Kind() == protoreflect.MessageKind {
					flattened, err := flattenMsgForQuery(key, v.Message())
					if err != nil {
						rangeErr = fmt.Errorf("flatten msg for query %s: %w", fieldName, err)
						return false
					}
					for _, item := range flattened {
						if item.val == "" {
							continue
						}
						if usedFieldsPath.hasFullKey(item.key) {
							continue
						}
						query.Add(item.key, item.val)
					}
				} else {
					strVal, err := stringifyValue(v, fd.MapValue())
					if err != nil {
						rangeErr = fmt.Errorf("stringify value for map %s: %w", fieldName, err)
						return false
					}
					query.Add(key, strVal)
				}
				return true
			})
			if rangeErr != nil {
				return nil, fmt.Errorf("map range error: %w", rangeErr)
			}

		case fd.Kind() == protoreflect.MessageKind:
			flattened, err := flattenMsgForQuery(fieldName, val.Message())
			if err != nil {
				return nil, fmt.Errorf("flatten msg for query %s: %w", fieldName, err)
			}
			for _, item := range flattened {
				if item.val == "" {
					continue
				}
				if usedFieldsPath.hasFullKey(item.key) {
					continue
				}
				query.Add(item.key, item.val)
			}

		default:
			strVal, err := stringifyValue(val, fd)
			if err != nil {
				return nil, fmt.Errorf("stringify value for primitive %s: %w", fieldName, err)
			}
			query.Add(fieldName, strVal)
		}
	}

	return query, nil
}

type flattenItem struct {
	key string
	val string
}

// flattenMsgForQuery flattens a non-repeated message value under a given prefix.
func flattenMsgForQuery(prefix string, msg protoreflect.Message) ([]flattenItem, error) {
	var out []flattenItem

	fields := msg.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		var (
			fd  = fields.Get(i)
			val = msg.Get(fd)
		)

		if isZeroValue(val, fd) {
			continue
		}

		key := fmt.Sprintf("%s.%s", prefix, fd.JSONName())

		switch {
		case fd.IsList():
			if fd.Kind() == protoreflect.MessageKind {
				return nil, fmt.Errorf("repeated message field %s cannot be flattened for query", key)
			}
			list := val.List()
			for j := 0; j < list.Len(); j++ {
				strVal, err := stringifyValue(list.Get(j), fd)
				if err != nil {
					return nil, fmt.Errorf("stringify query %s: %w", key, err)
				}
				out = append(out, flattenItem{key: key, val: strVal})
			}

		case fd.Kind() == protoreflect.MessageKind:
			nested, err := flattenMsgForQuery(key, val.Message())
			if err != nil {
				return nil, fmt.Errorf("flatten msg for query %s: %w", key, err)
			}
			out = append(out, nested...)

		case fd.IsMap():
			var mapErr error
			val.Map().Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
				keyStr := k.String()

				if fd.MapValue().Kind() == protoreflect.MessageKind {
					child, err := flattenMsgForQuery(keyStr, v.Message())
					if err != nil {
						mapErr = fmt.Errorf("flatten map value %s: %w", key, err)
						return false
					}
					out = append(out, child...)
				} else {
					strVal, err := stringifyValue(v, fd.MapValue())
					if err != nil {
						mapErr = fmt.Errorf("stringify query %s: %w", keyStr, err)
						return false
					}
					out = append(out, flattenItem{key: keyStr, val: strVal})
				}
				return true
			})
			if mapErr != nil {
				return nil, mapErr
			}

		default:
			strVal, err := stringifyValue(val, fd)
			if err != nil {
				return nil, fmt.Errorf("stringify query %s: %w", key, err)
			}
			out = append(out, flattenItem{key: key, val: strVal})
		}
	}

	return out, nil
}
