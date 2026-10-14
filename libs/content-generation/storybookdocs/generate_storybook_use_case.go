package storybookdocs

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"lore-master/libs/content-generation/generatedfile"
	"lore-master/libs/content-generation/inputselection"
)

// workers is how many stories are rendered at once, each in a tab of the one browser.
const workers = 4

// Generate renders the Storybook spec.Source names and writes its component pages. spec.Input,
// when given, selects stories by title with gitignore-style patterns ("Components/", "!Internal/").
func Generate(ctx context.Context, workspaceRoot string, spec generatedfile.Spec) (generatedfile.Output, error) {
	return generate(ctx, workspaceRoot, spec, func(ctx context.Context) (storyRenderer, error) {
		executable, err := findBrowser(systemLocator())
		if err != nil {
			return nil, err
		}

		browser, err := startBrowser(ctx, executable)
		if err != nil {
			return nil, err
		}

		return browser, nil
	})
}

// generate is Generate with the browser supplied, so it can be tested without one.
func generate(ctx context.Context, workspaceRoot string, spec generatedfile.Spec, start func(context.Context) (storyRenderer, error)) (generatedfile.Output, error) {
	if strings.TrimSpace(spec.Source) == "" {
		return generatedfile.Output{}, fmt.Errorf("storybook-docs needs a source: the URL of a Storybook, or the folder of its build (e.g. storybook-static)")
	}
	opened, err := openStorybook(workspaceRoot, spec.Source)
	if err != nil {
		return generatedfile.Output{}, err
	}
	defer opened.close()

	data, err := opened.readIndex(ctx)
	if err != nil {
		return generatedfile.Output{}, err
	}
	all, err := ParseStoryIndex(data)
	if err != nil {
		return generatedfile.Output{}, err
	}
	selector := inputselection.NewSelector(spec.Input, nil)
	var stories []Story
	for _, story := range all {
		if selector.Selects(story.Title, false) || selector.Selects(story.Title, true) {
			stories = append(stories, story)
		}
	}
	if len(stories) == 0 {
		return generatedfile.Output{Warnings: []string{"no story matched the input"}}, nil
	}

	renderer, err := start(ctx)
	if err != nil {
		return generatedfile.Output{}, err
	}
	defer renderer.Close()

	rendered := renderAll(ctx, renderer, opened.base, stories)
	if err := ctx.Err(); err != nil {
		return generatedfile.Output{}, err
	}

	title := spec.Title
	if title == "" {
		title = DefaultTitle
	}
	files, warnings := ComponentPages(rendered, title, spec.Source)

	return generatedfile.Output{Files: files, Warnings: warnings}, nil
}

// renderAll renders every story with a bounded number at a time and answers them in the
// order of stories, so the pages do not depend on which finished first.
func renderAll(ctx context.Context, renderer storyRenderer, base string, stories []Story) []renderedStory {
	results := make([]renderedStory, len(stories))
	queue := make(chan int)
	var group sync.WaitGroup
	for range min(workers, len(stories)) {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range queue {
				results[index] = renderer.Render(ctx, base, stories[index])
			}
		}()
	}
	for index := range stories {
		queue <- index
	}
	close(queue)
	group.Wait()

	return results
}
