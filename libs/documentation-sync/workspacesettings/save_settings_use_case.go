package workspacesettings

import (
	"bytes"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// newFileHeader opens a file written from scratch, so whoever opens it next learns what
// it is without looking anything up.
const newFileHeader = `LoreMaster configuration. Commit this file; it never holds a secret
(the editor keeps credentials in its own secret store).
Planned values, accepted later: content types test-results and code-docs (#96),
custom templates (#97).`

// SaveSettings writes settings back. When the file existed, the new values are merged
// into its YAML tree, so the author's comments and key order survive; a new file gets
// an explanatory header. Settings are validated first, the file is written through a
// temporary file and a rename, with LF line endings and a trailing newline.
func SaveSettings(loaded Loaded, settings Settings) error {
	if err := Validate(settings); err != nil {
		return err
	}
	var fresh yaml.Node
	if err := fresh.Encode(settings); err != nil {
		return err
	}
	document := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{&fresh}}
	if loaded.Exists && len(loaded.tree) > 0 {
		var existing yaml.Node
		if err := yaml.Unmarshal(loaded.tree, &existing); err != nil {
			return err
		}
		mergeInto(&existing, document)
		document = &existing
	} else {
		fresh.HeadComment = newFileHeader
	}

	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(document); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}

	return writeAtomically(loaded.path, out.Bytes())
}

func writeAtomically(path string, content []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temporary.Name()) }()
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()

		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}

	return os.Rename(temporary.Name(), path)
}
