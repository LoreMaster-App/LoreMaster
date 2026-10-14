package formatdetection

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"lore-master/libs/content-generation/generatedfile"
	"lore-master/libs/content-generation/inputselection"
)

// headBytes is how much of a file is looked at: enough for a root element or a version key.
const headBytes = 64 * 1024

// maxFiles bounds how many files are looked at, so a pattern that selects a whole repository
// fails fast instead of reading all of it.
const maxFiles = 5000

// skippedFolders are never searched: dependencies and version control.
var skippedFolders = []string{"node_modules", ".git"}

// candidateExtensions are the files worth opening.
var candidateExtensions = []string{".xml", ".info", ".lcov", ".md", ".markdown", ".yaml", ".yml", ".json"}

// Detection is what was recognised.
type Detection struct {
	// Type is the generator type that reads the files.
	Type string
	// Source is set when the format is read from a folder or URL, not from files: the folder
	// of a Storybook build found among the inputs.
	Source string
}

// Detect recognises the format of what spec selects. A spec with a Source is a Storybook. Else
// the files selected by spec.Input are opened and recognised one by one; it is an error when
// none is a format we read, and when they are several, since one generator writes one kind of
// page: the message says what was found so each can have its own generator.
func Detect(ctx context.Context, workspaceRoot string, spec generatedfile.Spec) (Detection, error) {
	if strings.TrimSpace(spec.Source) != "" {
		return Detection{Type: Storybook, Source: spec.Source}, nil
	}
	if len(spec.Input) == 0 {
		return Detection{}, fmt.Errorf("the type is not set and there is no input to recognise it from: name the files the command writes (input), or set a type")
	}

	found, examined, err := recognise(ctx, workspaceRoot, spec)
	if err != nil {
		return Detection{}, err
	}
	switch len(found) {
	case 0:
		return Detection{}, fmt.Errorf("could not identify the output, no parser available: %s. The formats we read are %s; check the input patterns, or set the type", describeExamined(examined), strings.Join(Formats, ", "))
	case 1:
		for kind, files := range found {
			if kind != Storybook {
				return Detection{Type: kind}, nil
			}
			folders := folderSet(files)
			if len(folders) > 1 {
				return Detection{}, fmt.Errorf("the input selects several Storybooks (%s); narrow it to one", strings.Join(folders, ", "))
			}

			return Detection{Type: Storybook, Source: folders[0]}, nil
		}
	}

	var parts []string
	for _, kind := range Formats {
		if files, ok := found[kind]; ok {
			parts = append(parts, fmt.Sprintf("%s (%s)", kind, files[0]))
		}
	}

	return Detection{}, fmt.Errorf("the input selects several formats: %s; give each its own generator with its own input, or set a type", strings.Join(parts, ", "))
}

// recognise lists, per recognised format, the workspace-relative files that are in it.
func recognise(ctx context.Context, workspaceRoot string, spec generatedfile.Spec) (map[string][]string, []string, error) {
	selector := inputselection.NewSelector(spec.Input, nil)
	output := strings.Trim(path.Clean(spec.Output), "/")
	found := map[string][]string{}
	opened := 0
	var examined []string

	err := filepath.WalkDir(workspaceRoot, func(full string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		relative, err := filepath.Rel(workspaceRoot, full)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			if full != workspaceRoot && slices.Contains(skippedFolders, entry.Name()) {
				return filepath.SkipDir
			}
			// What the generator wrote last time is its result, never its input.
			if output != "" && output != "." && relative == output {
				return filepath.SkipDir
			}

			return nil
		}
		if !slices.Contains(candidateExtensions, strings.ToLower(path.Ext(relative))) || !selector.Selects(relative, false) {
			return nil
		}
		if opened++; opened > maxFiles {
			return fmt.Errorf("the input selects more than %d files; narrow it to what the command writes", maxFiles)
		}
		examined = append(examined, fmt.Sprintf("%s is %s", relative, Container(relative)))
		head, err := readHead(full)
		if err != nil {
			return nil //nolint:nilerr // an unreadable file is not a format, it is left out
		}
		if kind := Sniff(relative, head); kind != "" {
			found[kind] = append(found[kind], relative)
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return found, examined, nil
}

// describeExamined says what the detection looked at, for the message of an output nothing
// recognised: how many files the input selected and the first few of them.
func describeExamined(examined []string) string {
	if len(examined) == 0 {
		return "the input selects no file a report or document could be (.xml, .info, .lcov, .md, .yaml, .yml or .json)"
	}
	shown := examined
	if len(shown) > 3 {
		shown = shown[:3]
	}

	return fmt.Sprintf("none of the %d file(s) it selects holds a schema we have a parser for (%s)", len(examined), strings.Join(shown, ", "))
}

func readHead(full string) ([]byte, error) {
	file, err := os.Open(full)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	head := make([]byte, headBytes)
	n, err := io.ReadFull(file, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}

	return head[:n], nil
}

// folderSet lists the distinct folders of files, workspace-relative; "." for the workspace.
func folderSet(files []string) []string {
	seen := map[string]bool{}
	var folders []string
	for _, file := range files {
		folder := path.Dir(file)
		if !seen[folder] {
			seen[folder] = true
			folders = append(folders, folder)
		}
	}
	slices.Sort(folders)

	return folders
}
