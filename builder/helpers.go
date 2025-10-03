package builder

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// findFieldByPath resolves a dot-separated field path in a protobuf message and returns the protoreflect value and its descriptor.
func findFieldByPath(msg protoreflect.Message, fieldPath string) (protoreflect.Value, protoreflect.FieldDescriptor, bool) {
	var (
		current    = msg
		parts      = strings.Split(fieldPath, ".")
		partsCount = len(parts) - 1
	)

	for i, part := range parts {
		fd, ok := findFieldByName(current, part)
		if !ok {
			return protoreflect.Value{}, nil, false
		}

		val := current.Get(fd)
		if i == partsCount { // it's last part
			return val, fd, true
		}

		if fd.Kind() != protoreflect.MessageKind {
			return protoreflect.Value{}, nil, false
		}
		current = val.Message()
	}

	return protoreflect.Value{}, nil, false
}

// findFieldByName find a field name in a protobuf message and returns the protoreflect field descriptor.
func findFieldByName(msg protoreflect.Message, fieldName string) (protoreflect.FieldDescriptor, bool) {
	fields := msg.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if fd.JSONName() == fieldName {
			return fd, true
		}
	}
	return nil, false
}

// isZeroValue checks if protoreflect.Value is zero for the field.
func isZeroValue(val protoreflect.Value, fd protoreflect.FieldDescriptor) bool {
	if fd.IsList() {
		return val.List().Len() == 0
	}
	if fd.IsMap() {
		return val.Map().Len() == 0
	}

	switch fd.Kind() {
	case protoreflect.StringKind:
		return val.String() == ""
	case protoreflect.BytesKind:
		return len(val.Bytes()) == 0
	case protoreflect.BoolKind:
		return !val.Bool()
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return val.Int() == 0
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return val.Uint() == 0
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return val.Float() == 0
	case protoreflect.EnumKind:
		return val.Enum() == 0
	case protoreflect.MessageKind:
		return !val.Message().IsValid()
	default:
		return !val.IsValid()
	}
}

// stringifyValue converts protoreflect.Value to string for path/query substitution.
func stringifyValue(val protoreflect.Value, fd protoreflect.FieldDescriptor) (string, error) {
	switch fd.Kind() {
	case protoreflect.StringKind:
		return val.String(), nil
	case protoreflect.BytesKind:
		return base64.StdEncoding.EncodeToString(val.Bytes()), nil
	case protoreflect.BoolKind:
		if val.Bool() {
			return "true", nil
		}
		return "false", nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return fmt.Sprintf("%d", val.Int()), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return fmt.Sprintf("%d", val.Uint()), nil
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return strconv.FormatFloat(val.Float(), 'g', -1, 64), nil
	case protoreflect.EnumKind:
		ed := fd.Enum().Values().ByNumber(val.Enum())
		if ed != nil {
			return string(ed.Name()), nil
		}
		return fmt.Sprintf("%d", val.Enum()), nil
	default:
		return "", fmt.Errorf("unsupported field kind: %s", fd.Kind())
	}
}
