package builder

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// -------------------------- Path template representation -------------------------

// pathSegment is a helper interface for elements of a path.
type pathSegment interface {
	isSegment() bool
}

// pathTemplate represents a parsed URL path template.
type pathTemplate struct {
	// literalPrefix is the fixed part of the path before the first {var},
	// e.g. "/v1/users/" for "/v1/users/{user_id}/orders:get".
	// It is removed from segments, so segments contain only the remaining path literals and variables.
	literalPrefix string

	// segments is a sequence of pathLiteral or pathVar representing the rest of the path after literalPrefix.
	segments []pathSegment

	// customVerb is an optional ":verb" suffix, e.g. ":get".
	customVerb string
}

// pathLiteral represents a fixed literal segment in a path template, e.g., "/v1/users/".
type pathLiteral struct {
	text string
}

func (p pathLiteral) isSegment() bool { return true }

// pathVar represents a variable segment in a path template, e.g., "{user.id}".
type pathVar struct {
	// fieldPath is the dotted path to the field in the struct, e.g., "user.id".
	fieldPath string

	// pattern is the optional pattern after '=', e.g., "*" or "**/orders".
	// It specifies how the variable can match parts of the URL path.
	pattern string

	// multiSegment is true if the pattern can match multiple path segments
	// (contains '/' or "**").
	multiSegment bool
}

func (p pathVar) isSegment() bool { return true }

// ----------------------------- Path template parsing -----------------------------

// parsePathTemplate parses a URL path template into a pathTemplate.
// It extracts:
//  1. literalPrefix — fixed part before the first variable,
//  2. segments — sequence of pathLiteral and pathVar,
//  3. customVerb — optional ":verb" suffix.
//
// Complexity: time O(n), memory O(n).
//
// Example:
//
//	input: "/v1/users/{user_id}/orders:get"
//	output: pathTemplate{
//	  literalPrefix: "/v1/users/",
//	  segments:      [{user_id}, "/orders"],
//	  customVerb:    ":get",
//	}
func parsePathTemplate(input string) (*pathTemplate, error) {
	// Step 1: extract custom verb after the last colon, e.g. ":get"
	var customVerb string
	if i := strings.LastIndex(input, ":"); i >= 0 && i > strings.LastIndex(input, "/") {
		customVerb = input[i:]
		input = input[:i]
	}

	var (
		segments []pathSegment
		buf      strings.Builder
	)

	// Step 2: iterate over the input and split into segments
	for i := 0; i < len(input); {
		if input[i] != '{' {
			buf.WriteByte(input[i])
			i++
			continue
		}

		// Add literal before '{' if any
		if buf.Len() > 0 {
			segments = append(segments, pathLiteral{text: buf.String()})
			buf.Reset()
		}

		// Find closing '}'
		start := i + 1
		offset := strings.IndexByte(input[start:], '}') // relative offset from start
		if offset < 0 {
			return nil, fmt.Errorf("unclosed '{' in path: %s", input)
		}
		end := start + offset

		token := input[start:end]
		i = end + 1 // jump past '}'

		// Split field path and optional pattern
		var fieldPath, pattern string
		if k := strings.IndexByte(token, '='); k >= 0 {
			fieldPath = strings.TrimSpace(token[:k])
			pattern = strings.TrimSpace(token[k+1:])
		} else {
			fieldPath = strings.TrimSpace(token)
		}

		if fieldPath == "" {
			return nil, fmt.Errorf("empty variable in path: %s", input)
		}

		pv := pathVar{
			fieldPath:    fieldPath,
			pattern:      pattern,
			multiSegment: isMultiSegmentPattern(pattern),
		}
		segments = append(segments, pv)
	}

	// Step 3: add any trailing literal after last '}'
	if buf.Len() > 0 {
		segments = append(segments, pathLiteral{text: buf.String()})
	}

	// Step 4: extract literalPrefix if the first segment is a literal
	var literalPrefix string
	if len(segments) > 0 {
		if pl, ok := segments[0].(pathLiteral); ok {
			literalPrefix = pl.text
			segments = segments[1:] // remove from segments to avoid duplication
		}
	}

	// Step 5: return fully parsed pathTemplate
	return &pathTemplate{
		literalPrefix: literalPrefix,
		segments:      segments,
		customVerb:    customVerb,
	}, nil
}

// isMultiSegmentPattern returns true if pattern can match multiple path segments (contains '/' or '**').
// Examples:
// | 	Pattern     | Result | 					Usecase					|
// |----------------|--------|------------------------------------------|
// | ""             | false  | {var} 			=> single segment		|
// | "*"            | false  | {var=*} 			=> single segment		|
// | "**"           | true   | {var=**} 		=> multiple segments	|
// | "foo/*"        | true   | {var=foo/*} 		=> multiple segments	|
// | "foo/**"       | true   | {var=foo/**} 	=> multiple segments	|
// | "users/*/orders"| true  | {users/*/orders} => multiple segments	|
func isMultiSegmentPattern(pattern string) bool {
	if pattern == "" {
		return false
	}
	if pattern == singleWildcard {
		return false
	}
	return strings.Contains(pattern, "/") || strings.Contains(pattern, doubleWildcard)
}

// ----------------------------- Path template resolving -----------------------------

// resolvePathPlaceholders expands placeholders in a path template using values from proto.Message.
// Placeholders must be bound to non-repeated scalar fields (not lists, maps, or messages).
//
// Example:
//
//	tmpl:       "/v1/users/{user_id}/orders:get"
//	msg:        &pb.Message{UserId: 12345}
//
//	path:       "/v1/users/12345/orders:get"
//	usedFields: {"user_id"}
func resolvePathPlaceholders(tmpl *pathTemplate, msg proto.Message) (path string, usedFields *usedFields, err error) {
	usedFields = newUsedFields()

	var sb strings.Builder
	sb.WriteString(tmpl.literalPrefix)

	msgReflect := msg.ProtoReflect()

	for _, segment := range tmpl.segments {
		switch s := segment.(type) {
		case pathLiteral:
			sb.WriteString(s.text)

		case pathVar:
			val, fd, ok := findFieldByPath(msgReflect, s.fieldPath)
			if !ok {
				return "", nil, fmt.Errorf("path placeholder %s not found", s.fieldPath)
			}
			if isZeroValue(val, fd) {
				// it's the only case that allows zero-value matches.
				if s.pattern == doubleWildcard {
					usedFields.add(s.fieldPath)
					continue
				}
				return "", nil, fmt.Errorf("path placeholder %s has zero value", s.fieldPath)
			}

			// must be scalar (non-repeated, non-map, non-message)
			if fd.IsList() || fd.IsMap() || fd.Kind() == protoreflect.MessageKind {
				return "", nil, fmt.Errorf("path placeholder %s must be scalar", s.fieldPath)
			}

			usedFields.add(s.fieldPath)

			var strVal string
			strVal, err = stringifyValue(val, fd)
			if err != nil {
				return "", nil, fmt.Errorf("stringify placeholder %s: %w", s.fieldPath, err)
			}

			if err = validatePattern(s.pattern, strVal); err != nil {
				return "", nil, fmt.Errorf("validate pattern, %s:%s: %w", s.fieldPath, strVal, err)
			}

			parts := strings.Split(strVal, "/")
			for i := range parts {
				parts[i] = url.PathEscape(parts[i])
			}
			sb.WriteString(strings.Join(parts, "/"))
		}
	}

	sb.WriteString(tmpl.customVerb)
	return sb.String(), usedFields, nil
}

// validatePattern checks whether input matches the given path pattern.
//
// Rules:
//   - "" or "*" => exactly one segment, no "/" allowed
//   - "**"      => zero or more segments (may include "/")
//   - composite patterns like "*/orders/*" must match literally
//
// Example for composite pattern case:
//
//	pattern: "*/orders/*"
//	input:   "42/orders/123"
//
//	patternSegments = ["*", "orders", "*"]
//	valueParts      = ["42", "orders", "123"]
//
//	Match:
//	  "*"      -> "42"
//	  "orders" -> "orders"
//	  "*"      -> "123"
func validatePattern(pattern, input string) error {
	var (
		parts    = strings.Split(input, "/")
		lenParts = len(parts)
	)

	if pattern == "" || pattern == singleWildcard {
		if lenParts != 1 {
			return errors.New("must be a single path segment")
		}
		return nil
	}

	if pattern == doubleWildcard {
		if lenParts < 1 {
			return errors.New("must contain at least one segment")
		}
		return nil
	}

	var (
		patternSegments = strings.Split(pattern, "/")
		patternIndex    int
	)

	for i := 0; i < len(patternSegments); i++ {
		switch patternSegments[i] {
		case singleWildcard:
			if patternIndex >= lenParts || parts[patternIndex] == "" {
				return fmt.Errorf("segment %d must not be empty", patternIndex)
			}
			patternIndex++
		case doubleWildcard:
			if patternIndex >= lenParts {
				return fmt.Errorf("must contain at least one segment at position %d", patternIndex)
			}
			return nil
		default:
			if patternIndex >= lenParts || parts[patternIndex] != patternSegments[i] {
				return fmt.Errorf("expected literal %s at position %d", patternSegments[i], patternIndex)
			}
			patternIndex++
		}
	}

	if patternIndex != lenParts {
		return errors.New("extra segments in value")
	}

	return nil
}
