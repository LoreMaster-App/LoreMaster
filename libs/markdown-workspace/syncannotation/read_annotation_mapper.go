package syncannotation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

var byteOrderMark = []byte{0xEF, 0xBB, 0xBF}

// Read splits a Markdown file into its annotation and its body. A file without an
// annotation is not an error. An annotation that cannot be trusted (unclosed, a line
// that is not "key: value", a repeated key, a malformed typed value) is: syncing from it
// could update the wrong page or create a duplicate.
func Read(content []byte) (Document, error) {
	layout := Layout{ByteOrderMark: bytes.HasPrefix(content, byteOrderMark), LineEnding: lineEndingOf(content)}
	rest := bytes.TrimPrefix(content, byteOrderMark)
	document := Document{Body: rest, Layout: layout}

	start := frontMatterEnd(rest)
	if line, _ := lineAt(rest, start); strings.TrimRight(line, " \t") != OpeningLine {
		return document, nil
	}

	firstLine := 1 + bytes.Count(rest[:start], []byte("\n"))
	closing := -1
	for offset := nextLine(rest, start); offset < len(rest); offset = nextLine(rest, offset) {
		if line, _ := lineAt(rest, offset); strings.TrimSpace(line) == ClosingLine {
			closing = offset

			break
		}
	}
	if closing < 0 {
		return Document{}, fmt.Errorf("the lore-master annotation opened on line %d is never closed with %q", firstLine, ClosingLine)
	}

	var fields []Field
	lineNumber := firstLine
	for offset := nextLine(rest, start); offset < closing; offset = nextLine(rest, offset) {
		lineNumber++
		line, _ := lineAt(rest, offset)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		key, value, isField := strings.Cut(trimmed, ":")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if !isField || key == "" || strings.ContainsAny(key, " \t") {
			return Document{}, fmt.Errorf("line %d of the lore-master annotation is not \"key: value\": %q", lineNumber, trimmed)
		}
		if slices.ContainsFunc(fields, func(f Field) bool { return f.Key == key }) {
			return Document{}, fmt.Errorf("line %d of the lore-master annotation repeats the key %q", lineNumber, key)
		}
		fields = append(fields, Field{Key: key, Value: value})
	}
	document.Body = withoutBlock(rest, start, closing, nextLine(rest, closing))

	annotation, warnings, err := annotationFrom(fields)
	if err != nil {
		return Document{}, err
	}
	document.Annotation = &annotation
	document.Warnings = warnings

	return document, nil
}

func annotationFrom(fields []Field) (Annotation, []string, error) {
	var annotation Annotation
	var warnings []string
	for _, field := range fields {
		var err error
		switch field.Key {
		case KeyPlatform:
			annotation.Platform = field.Value
		case KeyBaseURL:
			annotation.BaseURL = field.Value
		case KeySpace:
			annotation.Space = field.Value
		case KeyPageID:
			annotation.PageID = field.Value
		case KeyParentID:
			annotation.ParentID = field.Value
		case KeyVersion:
			annotation.Version, err = strconv.Atoi(field.Value)
		case KeyContentHash:
			annotation.ContentHash = field.Value
		case KeyAttachments:
			err = json.Unmarshal([]byte(field.Value), &annotation.Attachments)
		case KeySyncedAt:
			annotation.SyncedAt, err = time.Parse(time.RFC3339, field.Value)
		case KeyTitle:
			annotation.Title = field.Value
		case KeyParent:
			annotation.Parent = field.Value
		default:
			annotation.Unknown = append(annotation.Unknown, field)
			warnings = append(warnings, fmt.Sprintf("the lore-master annotation has an unknown key %q; it is kept as is", field.Key))
		}
		if err != nil {
			return Annotation{}, nil, fmt.Errorf("the lore-master annotation's %q value %q is invalid: %w", field.Key, field.Value, err)
		}
	}

	return annotation, warnings, nil
}

// withoutBlock removes the block that opens at start and whose closing line runs from
// closing to next. The line ending after the block goes with it; when the block ends
// the file with no line ending of its own, the one before it goes instead, which is
// exactly what the writer adds in that position.
func withoutBlock(rest []byte, start int, closing int, next int) []byte {
	endsWithLineEnding := next > closing && rest[next-1] == '\n'
	if !endsWithLineEnding && next == len(rest) && start > 0 {
		start -= len(lineEndingBefore(rest, start))
	}

	return slices.Concat(rest[:start], rest[next:])
}

// frontMatterEnd is the offset just past a YAML front matter block that opens the
// file ("---" on line one, closed by "---" or "..."), or 0 when there is none.
func frontMatterEnd(content []byte) int {
	if line, _ := lineAt(content, 0); strings.TrimRight(line, " \t") != "---" {
		return 0
	}
	for offset := nextLine(content, 0); offset < len(content); {
		line, next := lineAt(content, offset)
		if fence := strings.TrimRight(line, " \t"); fence == "---" || fence == "..." {
			return next
		}
		offset = next
	}

	return 0
}

// lineAt returns the line starting at offset without its line ending, and the offset
// of the line after it.
func lineAt(content []byte, offset int) (string, int) {
	if offset >= len(content) {
		return "", len(content)
	}
	end := bytes.IndexByte(content[offset:], '\n')
	if end < 0 {
		return string(content[offset:]), len(content)
	}

	return strings.TrimSuffix(string(content[offset:offset+end]), "\r"), offset + end + 1
}

func nextLine(content []byte, offset int) int {
	_, next := lineAt(content, offset)

	return next
}

func lineEndingOf(content []byte) string {
	end := bytes.IndexByte(content, '\n')
	if end > 0 && content[end-1] == '\r' {
		return "\r\n"
	}

	return "\n"
}

func lineEndingBefore(content []byte, offset int) string {
	if offset >= 2 && content[offset-2] == '\r' && content[offset-1] == '\n' {
		return "\r\n"
	}
	if offset >= 1 && content[offset-1] == '\n' {
		return "\n"
	}

	return ""
}
