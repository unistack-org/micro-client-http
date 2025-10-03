package builder

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func buildSingleFieldBody(msg proto.Message, fieldName string) (proto.Message, error) {
	msgReflect := msg.ProtoReflect()

	fd, found := findFieldByName(msgReflect, fieldName)
	if !found || fd == nil {
		return nil, fmt.Errorf("field %s not found", fieldName)
	}
	if !msgReflect.Has(fd) {
		return nil, fmt.Errorf("field %s is not set", fieldName)
	}

	val := msgReflect.Get(fd)

	if fd.Kind() == protoreflect.MessageKind {
		return val.Message().Interface(), nil
	}

	newMsg := proto.Clone(msg)
	newMsgReflect := newMsg.ProtoReflect()
	newMsgReflect.Range(func(f protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		if f != fd {
			newMsgReflect.Clear(f)
		}
		return true
	})

	return newMsg, nil
}

func buildFullBody(msg proto.Message, usedFieldsPath *usedFields) (proto.Message, error) {
	var (
		msgReflect    = msg.ProtoReflect()
		newMsg        = msgReflect.New().Interface()
		newMsgReflect = newMsg.ProtoReflect()
	)

	fields := msgReflect.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		fieldName := fd.JSONName()

		if usedFieldsPath.hasTopLevelKey(fieldName) {
			continue
		}

		val := msgReflect.Get(fd)
		if !val.IsValid() {
			continue
		}

		// Note: order of the cases is important!
		switch {
		case fd.IsList():
			list := val.List()
			newList := newMsgReflect.Mutable(fd).List()

			if fd.Kind() == protoreflect.MessageKind {
				for j := 0; j < list.Len(); j++ {
					elem, err := buildFullBody(list.Get(j).Message().Interface(), usedFieldsPath)
					if err != nil {
						return nil, fmt.Errorf("recursive build full body: %w", err)
					}
					newList.Append(protoreflect.ValueOfMessage(elem.ProtoReflect()))
				}
			} else {
				for j := 0; j < list.Len(); j++ {
					newList.Append(list.Get(j))
				}
			}

		case fd.IsMap():
			var (
				m        = val.Map()
				newMap   = newMsgReflect.Mutable(fd).Map()
				rangeErr error
			)
			m.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
				if fd.MapValue().Kind() == protoreflect.MessageKind {
					elem, err := buildFullBody(v.Message().Interface(), usedFieldsPath)
					if err != nil {
						rangeErr = fmt.Errorf("recursive build full body: %w", err)
						return false
					}
					newMap.Set(k, protoreflect.ValueOfMessage(elem.ProtoReflect()))
				} else {
					newMap.Set(k, v)
				}
				return true
			})
			if rangeErr != nil {
				return nil, fmt.Errorf("map range error: %w", rangeErr)
			}

		case fd.Kind() == protoreflect.MessageKind:
			elem, err := buildFullBody(val.Message().Interface(), usedFieldsPath)
			if err != nil {
				return nil, fmt.Errorf("recursive build full body: %w", err)
			}
			newMsgReflect.Set(fd, protoreflect.ValueOfMessage(elem.ProtoReflect()))

		default:
			newMsgReflect.Set(fd, val)
		}
	}

	return newMsg, nil
}
