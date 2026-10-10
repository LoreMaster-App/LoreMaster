package workspacesettings

import "gopkg.in/yaml.v3"

// mergeInto writes the values of updated over existing, keeping what makes the file
// the author's: comments on every key and value, key order, and quoting style where the
// value is unchanged. Keys only in updated are appended; keys only in existing are kept.
// Sequences are matched by position.
func mergeInto(existing *yaml.Node, updated *yaml.Node) {
	if existing.Kind == yaml.DocumentNode && updated.Kind == yaml.DocumentNode && len(existing.Content) > 0 && len(updated.Content) > 0 {
		mergeInto(existing.Content[0], updated.Content[0])

		return
	}
	if existing.Kind != updated.Kind {
		replaceKeepingComments(existing, updated)

		return
	}
	switch existing.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(updated.Content); i += 2 {
			key, value := updated.Content[i], updated.Content[i+1]
			if found := valueOf(existing, key.Value); found != nil {
				mergeInto(found, value)
			} else {
				existing.Content = append(existing.Content, key, value)
			}
		}
	case yaml.SequenceNode:
		for i, item := range updated.Content {
			if i < len(existing.Content) {
				mergeInto(existing.Content[i], item)
			} else {
				existing.Content = append(existing.Content, item)
			}
		}
		existing.Content = existing.Content[:len(updated.Content)]
	case yaml.ScalarNode:
		if existing.Value != updated.Value || existing.Tag != updated.Tag {
			existing.Value, existing.Tag = updated.Value, updated.Tag
			existing.Style = updated.Style
		}
	}
}

func valueOf(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}

	return nil
}

func replaceKeepingComments(existing *yaml.Node, updated *yaml.Node) {
	head, line, foot := existing.HeadComment, existing.LineComment, existing.FootComment
	*existing = *updated
	existing.HeadComment, existing.LineComment, existing.FootComment = head, line, foot
}

// optionalKeys are the keys the settings leave out when they are empty, by the path of the
// mapping that holds them. mergeInto keeps keys that are only in the file, which is right for
// keys this version does not know, but an optional key this version does know that is missing
// from the update was cleared, and must go.
var optionalKeys = map[string][]string{
	"":                {"skipGitignored", "ignore", "generators"},
	"outputs":         {"repo", "branch", "include", "exclude"},
	"outputs.content": {"excludes"},
	"generators":      {"input", "title"},
}

// dropClearedKeys removes from existing the optional keys that updated no longer has, walking
// the two trees together; sequences of mappings are matched by position, as mergeInto does.
func dropClearedKeys(existing *yaml.Node, updated *yaml.Node, path string) {
	if existing.Kind == yaml.DocumentNode && updated.Kind == yaml.DocumentNode && len(existing.Content) > 0 && len(updated.Content) > 0 {
		dropClearedKeys(existing.Content[0], updated.Content[0], path)

		return
	}
	if existing.Kind != yaml.MappingNode || updated.Kind != yaml.MappingNode {
		return
	}

	for _, key := range optionalKeys[path] {
		if valueOf(existing, key) != nil && valueOf(updated, key) == nil {
			removeKey(existing, key)
		}
	}
	for i := 0; i+1 < len(existing.Content); i += 2 {
		key := existing.Content[i].Value
		current, replacement := existing.Content[i+1], valueOf(updated, key)
		if replacement == nil || current.Kind != yaml.SequenceNode || replacement.Kind != yaml.SequenceNode {
			continue
		}
		child := key
		if path != "" {
			child = path + "." + key
		}
		for j := 0; j < len(current.Content) && j < len(replacement.Content); j++ {
			dropClearedKeys(current.Content[j], replacement.Content[j], child)
		}
	}
}

// removeKey deletes a key and its value from a mapping.
func removeKey(mapping *yaml.Node, key string) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)

			return
		}
	}
}
