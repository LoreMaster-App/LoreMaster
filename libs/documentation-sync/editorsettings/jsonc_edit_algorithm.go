package editorsettings

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// property is where one top-level entry of a settings.json sits in the text.
type property struct {
	key                  string
	keyStart             int
	valueStart, valueEnd int
	commaAfter           int // index just past the comma that follows the value, or -1
}

// setKey puts value (already JSON) under key in the top-level object of a JSONC document,
// leaving every comment and every other key exactly as it was: the value of an existing
// key is replaced in place, a new key is added after the last one. A nil value removes the
// key. An empty document becomes a new object.
func setKey(source []byte, key string, value []byte) ([]byte, error) {
	if len(bytes.TrimSpace(source)) == 0 {
		if value == nil {
			return source, nil
		}

		return []byte("{\n  " + quote(key) + ": " + indentValue(value) + "\n}\n"), nil
	}
	properties, closeBrace, err := scanProperties(source)
	if err != nil {
		return nil, err
	}
	for _, p := range properties {
		if p.key != key {
			continue
		}
		if value != nil {
			return splice(source, p.valueStart, p.valueEnd, []byte(indentValue(value))), nil
		}
		from := p.keyStart
		for from > 0 && (source[from-1] == ' ' || source[from-1] == '\t') {
			from--
		}
		to := p.valueEnd
		if p.commaAfter >= 0 {
			to = p.commaAfter
		}
		for to < len(source) && (source[to] == ' ' || source[to] == '\t') {
			to++
		}
		if to < len(source) && source[to] == '\r' {
			to++
		}
		if to < len(source) && source[to] == '\n' {
			to++
		}

		return splice(source, from, to, nil), nil
	}
	if value == nil {
		return source, nil
	}
	entry := quote(key) + ": " + indentValue(value)
	if len(properties) == 0 {
		return splice(source, closeBrace, closeBrace, []byte("\n  "+entry+"\n")), nil
	}
	last := properties[len(properties)-1]

	return splice(source, last.valueEnd, last.valueEnd, []byte(",\n  "+entry)), nil
}

func quote(s string) string {
	encoded, _ := json.Marshal(s)

	return string(encoded)
}

// indentValue lays a JSON value out with two-space indentation, nested one level.
func indentValue(value []byte) string {
	var out bytes.Buffer
	if err := json.Indent(&out, value, "  ", "  "); err != nil {
		return string(value)
	}

	return out.String()
}

func splice(source []byte, from, to int, replacement []byte) []byte {
	out := make([]byte, 0, len(source)-(to-from)+len(replacement))
	out = append(out, source[:from]...)
	out = append(out, replacement...)

	return append(out, source[to:]...)
}

// scanProperties walks the top-level object of a JSONC document.
func scanProperties(source []byte) (properties []property, closeBrace int, err error) {
	i := skipTrivia(source, 0)
	if i >= len(source) || source[i] != '{' {
		return nil, 0, fmt.Errorf("settings file is not a JSON object")
	}
	i = skipTrivia(source, i+1)
	for i < len(source) && source[i] != '}' {
		if source[i] != '"' {
			return nil, 0, fmt.Errorf("expected a property name at offset %d", i)
		}
		keyStart := i
		keyEnd := skipString(source, i)
		var key string
		if err := json.Unmarshal(source[keyStart:keyEnd], &key); err != nil {
			return nil, 0, err
		}
		i = skipTrivia(source, keyEnd)
		if i >= len(source) || source[i] != ':' {
			return nil, 0, fmt.Errorf("expected ':' after %q", key)
		}
		i = skipTrivia(source, i+1)
		valueStart := i
		valueEnd := skipValue(source, i)
		p := property{key: key, keyStart: keyStart, valueStart: valueStart, valueEnd: valueEnd, commaAfter: -1}
		i = skipTrivia(source, valueEnd)
		if i < len(source) && source[i] == ',' {
			p.commaAfter = i + 1
			i = skipTrivia(source, i+1)
		}
		properties = append(properties, p)
	}
	if i >= len(source) {
		return nil, 0, fmt.Errorf("settings file is not closed")
	}

	return properties, i, nil
}

func skipTrivia(source []byte, i int) int {
	for i < len(source) {
		switch {
		case source[i] == ' ' || source[i] == '\t' || source[i] == '\r' || source[i] == '\n':
			i++
		case source[i] == '/' && i+1 < len(source) && source[i+1] == '/':
			for i < len(source) && source[i] != '\n' {
				i++
			}
		case source[i] == '/' && i+1 < len(source) && source[i+1] == '*':
			i += 2
			for i+1 < len(source) && (source[i] != '*' || source[i+1] != '/') {
				i++
			}
			i += 2
		default:
			return i
		}
	}

	return i
}

// skipString returns the index just past the string that starts at i.
func skipString(source []byte, i int) int {
	i++
	for i < len(source) {
		switch source[i] {
		case '\\':
			i += 2
		case '"':
			return i + 1
		default:
			i++
		}
	}

	return i
}

// skipValue returns the index just past the value that starts at i.
func skipValue(source []byte, i int) int {
	if i >= len(source) {
		return i
	}
	switch source[i] {
	case '"':
		return skipString(source, i)
	case '{', '[':
		depth := 0
		for i < len(source) {
			switch source[i] {
			case '"':
				i = skipString(source, i)

				continue
			case '/':
				if next := skipTrivia(source, i); next != i {
					i = next

					continue
				}
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					return i + 1
				}
			}
			i++
		}

		return i
	default:
		for i < len(source) {
			c := source[i]
			if c == ',' || c == '}' || c == ']' || c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '/' {
				break
			}
			i++
		}

		return i
	}
}
