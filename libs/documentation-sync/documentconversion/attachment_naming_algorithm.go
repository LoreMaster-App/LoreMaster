package documentconversion

import (
	"path"
	"strconv"
	"strings"

	"lore-master/libs/markdown-workspace/documentdiscovery"
)

// nameAttachments gives each distinct file a name that is unique on the page. A file
// keeps its base name unless another file on the page shares it (in any case, since
// platforms often compare attachment names that way); then each of them takes as many
// of its parent directories as it needs, joined by "__": docs/img/diagram.png and
// api/img/diagram.png become docs__img__diagram.png and api__img__diagram.png.
func nameAttachments(paths []documentdiscovery.DocumentPath) []Attachment {
	var distinct []documentdiscovery.DocumentPath
	seen := map[documentdiscovery.DocumentPath]bool{}
	for _, file := range paths {
		if !seen[file] {
			seen[file] = true
			distinct = append(distinct, file)
		}
	}

	sharing := map[string]int{}
	for _, file := range distinct {
		sharing[strings.ToLower(path.Base(string(file)))]++
	}
	taken := map[string]bool{}
	names := make([]string, len(distinct))
	for i, file := range distinct {
		if sharing[strings.ToLower(path.Base(string(file)))] == 1 {
			names[i] = path.Base(string(file))
			taken[strings.ToLower(names[i])] = true
		}
	}
	for i, file := range distinct {
		if names[i] != "" {
			continue
		}
		segments := strings.Split(string(file), "/")
		// A file at the workspace root has no folder to add and keeps its name.
		name := path.Base(string(file))
		for depth := 2; depth <= len(segments); depth++ {
			name = strings.Join(segments[len(segments)-depth:], "__")
			if !taken[strings.ToLower(name)] && !sharesName(distinct, file, depth, name) {
				break
			}
		}
		for n := 2; taken[strings.ToLower(name)]; n++ {
			name = strconv.Itoa(n) + "__" + path.Base(string(file))
		}
		names[i] = name
		taken[strings.ToLower(name)] = true
	}

	attachments := make([]Attachment, len(distinct))
	for i, file := range distinct {
		attachments[i] = Attachment{Filename: names[i], Path: file}
	}

	return attachments
}

// sharesName reports whether another file would get the same name at this depth, so
// both go one directory further up.
func sharesName(files []documentdiscovery.DocumentPath, self documentdiscovery.DocumentPath, depth int, name string) bool {
	for _, other := range files {
		if other == self {
			continue
		}
		segments := strings.Split(string(other), "/")
		if len(segments) >= depth && strings.EqualFold(strings.Join(segments[len(segments)-depth:], "__"), name) {
			return true
		}
	}

	return false
}
