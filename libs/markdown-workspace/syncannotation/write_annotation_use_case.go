package syncannotation

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Write puts annotation into the file at path and reports whether the file changed. A
// file whose annotation already says the same thing is not touched at all, so a sync
// that changed nothing leaves no diff, whatever key order or spacing the author used.
func Write(path string, annotation Annotation) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	updated, err := Render(content, annotation)
	if err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	if slices.Equal(updated, content) {
		return false, nil
	}

	return true, replaceFile(path, updated)
}

// Render returns content with its annotation replaced by (or, when it has none, given)
// annotation. Everything else keeps its bytes: the byte-order mark, the line-ending
// style and the body, so the body's ContentHash is the same before and after.
func Render(content []byte, annotation Annotation) ([]byte, error) {
	document, err := Read(content)
	if err != nil {
		return nil, err
	}
	if document.Annotation != nil && reflect.DeepEqual(normalised(*document.Annotation), normalised(annotation)) {
		return content, nil
	}
	lines, err := blockLines(annotation)
	if err != nil {
		return nil, err
	}

	eol := document.Layout.LineEnding
	block := strings.Join(lines, eol)
	body := document.Body
	at := frontMatterEnd(body)
	var inserted string
	if at == len(body) && at > 0 && body[at-1] != '\n' {
		inserted = eol + block
	} else {
		inserted = block + eol
	}

	var prefix []byte
	if document.Layout.ByteOrderMark {
		prefix = byteOrderMark
	}

	return slices.Concat(prefix, body[:at], []byte(inserted), body[at:]), nil
}

func blockLines(annotation Annotation) ([]string, error) {
	values := map[string]string{
		KeyPlatform:    annotation.Platform,
		KeyBaseURL:     annotation.BaseURL,
		KeySpace:       annotation.Space,
		KeyPageID:      annotation.PageID,
		KeyParentID:    annotation.ParentID,
		KeyContentHash: annotation.ContentHash,
		KeyTitle:       annotation.Title,
		KeyParent:      annotation.Parent,
	}
	if annotation.Version != 0 {
		values[KeyVersion] = strconv.Itoa(annotation.Version)
	}
	if len(annotation.Attachments) > 0 {
		encoded, err := json.Marshal(annotation.Attachments)
		if err != nil {
			return nil, err
		}
		values[KeyAttachments] = string(encoded)
	}
	if !annotation.SyncedAt.IsZero() {
		values[KeySyncedAt] = annotation.SyncedAt.UTC().Format(time.RFC3339)
	}

	lines := []string{OpeningLine}
	add := func(key string, value string) error {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil
		}
		if strings.ContainsAny(value, "\r\n") || strings.Contains(value, ClosingLine) {
			return fmt.Errorf("the annotation's %q value cannot contain a line break or %q: %q", key, ClosingLine, value)
		}
		lines = append(lines, key+": "+value)

		return nil
	}
	var errs []error
	for _, key := range knownKeys {
		errs = append(errs, add(key, values[key]))
	}
	for _, field := range annotation.Unknown {
		if slices.Contains(knownKeys, field.Key) || field.Key == "" || strings.ContainsAny(field.Key, ": \t\r\n") {
			errs = append(errs, fmt.Errorf("the annotation cannot carry %q as an unknown key", field.Key))

			continue
		}
		errs = append(errs, add(field.Key, field.Value))
	}

	return append(lines, ClosingLine), errors.Join(errs...)
}

// normalised is the annotation as it reads back after being written, so two
// annotations compare equal exactly when writing one over the other is a no-op.
func normalised(annotation Annotation) Annotation {
	trim := strings.TrimSpace
	annotation.Platform, annotation.BaseURL, annotation.Space = trim(annotation.Platform), trim(annotation.BaseURL), trim(annotation.Space)
	annotation.PageID, annotation.ParentID, annotation.ContentHash = trim(annotation.PageID), trim(annotation.ParentID), trim(annotation.ContentHash)
	annotation.Title, annotation.Parent = trim(annotation.Title), trim(annotation.Parent)
	if len(annotation.Attachments) == 0 {
		annotation.Attachments = nil
	}
	if !annotation.SyncedAt.IsZero() {
		annotation.SyncedAt = annotation.SyncedAt.UTC().Truncate(time.Second)
	} else {
		annotation.SyncedAt = time.Time{}
	}
	var unknown []Field
	for _, field := range annotation.Unknown {
		if value := trim(field.Value); value != "" {
			unknown = append(unknown, Field{Key: field.Key, Value: value})
		}
	}
	annotation.Unknown = unknown

	return annotation
}

// replaceFile writes through a temporary file in the same directory and renames it
// over the original, so an interrupted sync never leaves a half-written document.
func replaceFile(path string, content []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	// After a successful rename there is nothing left to remove; before it, the error
	// that stopped the write is the one worth reporting.
	defer func() { _ = os.Remove(temporary.Name()) }()
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()

		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Chmod(temporary.Name(), info.Mode().Perm()); err != nil {
		return err
	}

	return os.Rename(temporary.Name(), path)
}
