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
