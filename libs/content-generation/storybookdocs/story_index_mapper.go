package storybookdocs

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Story is one story of the Storybook's index.
type Story struct {
	ID   string
	Name string
	// Title is the component's place in the sidebar, "Components/Button": folders and a name.
	Title      string
	ImportPath string
}

// rawIndex reads both index shapes: entries (Storybook 7 and later) and stories (6.4).
type rawIndex struct {
	Entries map[string]rawEntry `json:"entries"`
	Stories map[string]rawEntry `json:"stories"`
}

type rawEntry struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Title      string `json:"title"`
	Kind       string `json:"kind"`
	Story      string `json:"story"`
	ImportPath string `json:"importPath"`
}

// ParseStoryIndex lists the stories of a Storybook's index.json (or stories.json), ordered by
// title and then id so the same Storybook always gives the same pages. Documentation entries
// are not stories and are left out.
func ParseStoryIndex(data []byte) ([]Story, error) {
	var index rawIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("the Storybook index is not valid JSON: %w", err)
	}
	entries := index.Entries
	if len(entries) == 0 {
		entries = index.Stories
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("the Storybook index lists no stories")
	}

	var stories []Story
	for key, entry := range entries {
		if entry.Type != "" && entry.Type != "story" {
			continue
		}
		story := Story{ID: entry.ID, Name: entry.Name, Title: entry.Title, ImportPath: entry.ImportPath}
		if story.ID == "" {
			story.ID = key
		}
		if story.Title == "" {
			story.Title = entry.Kind
		}
		if story.Name == "" {
			story.Name = entry.Story
		}
		story.Title = cleanTitle(story.Title)
		if story.Title == "" || story.Name == "" {
			continue
		}
		stories = append(stories, story)
	}
	slices.SortFunc(stories, func(a, b Story) int {
		if c := strings.Compare(a.Title, b.Title); c != 0 {
			return c
		}

		return strings.Compare(a.ID, b.ID)
	})

	return stories, nil
}

// cleanTitle trims the spaces around each folder of a title.
func cleanTitle(title string) string {
	parts := strings.Split(title, "/")
	kept := parts[:0]
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			kept = append(kept, part)
		}
	}

	return strings.Join(kept, "/")
}
