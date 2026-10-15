package syncstate

import (
	"fmt"

	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
	"lore-master/libs/markdown-workspace/syncannotation"
)

// Applied is what applying the state to the documents left behind.
type Applied struct {
	// Loaded holds each file's annotation exactly as the file carried it, before the state
	// was applied; the write-back needs it to migrate the old per-file state.
	Loaded map[documentdiscovery.DocumentPath]*syncannotation.Annotation
	// Warnings say why the state could not be read; the documents keep whatever annotation
	// their files carry.
	Warnings []string
}

// Apply gives each document the state recorded for it under key in lore-master.json, so
// planning sees the page a file became. A file the state does not know keeps the annotation
// it carries, which is how an old workspace is read before its first migrating sync.
func Apply(workspaceRoot string, key string, documents []documentparsing.MarkdownDocument) Applied {
	applied := Applied{Loaded: make(map[documentdiscovery.DocumentPath]*syncannotation.Annotation, len(documents))}
	for _, document := range documents {
		if document.Annotation != nil {
			copied := *document.Annotation
			applied.Loaded[document.Path] = &copied
		}
	}
	file, err := Read(workspaceRoot)
	if err != nil {
		applied.Warnings = append(applied.Warnings, fmt.Sprintf("the sync state was not applied: %v", err))

		return applied
	}
	storage := file.StorageNamed(key)
	for index := range documents {
		page, ok := storage.Pages[string(documents[index].Path)]
		if !ok {
			continue
		}
		documents[index].Annotation = AnnotationFor(documents[index].Annotation, storage, page)
	}

	return applied
}

// CarriesState reports whether a file's own annotation holds sync state for the target
// (baseURL and space; an annotation naming neither belongs to any target).
func CarriesState(annotation *syncannotation.Annotation, baseURL string, space string) bool {
	return annotation != nil && syncannotation.HasState(*annotation) && MatchesTarget(*annotation, baseURL, space)
}

// MatchesTarget reports whether the annotation was written by a sync to this site and space.
func MatchesTarget(annotation syncannotation.Annotation, baseURL string, space string) bool {
	return (annotation.BaseURL == "" || annotation.BaseURL == baseURL) && (annotation.Space == "" || annotation.Space == space)
}
