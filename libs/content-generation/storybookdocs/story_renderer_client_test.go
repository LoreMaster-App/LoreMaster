package storybookdocs

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakeStorybookPage draws what Storybook 7+ does: the sb-show-main class, a child of the root,
// and a preview whose store answers a story's argTypes and docs parameters.
const fakeStorybookPage = `<!doctype html><html><body class="sb-main-padded">
<div id="storybook-root"></div>
<script>
window.__STORYBOOK_PREVIEW__ = { storyStoreValue: { loadStory: async ({ storyId }) => ({
  argTypes: { label: { description: 'The text', table: { type: { summary: 'string' } } } },
  parameters: { docs: { source: { code: '<Label id="' + storyId + '" />' } } },
}) } };
setTimeout(() => {
  document.getElementById('storybook-root').innerHTML = '<div style="width:300px;height:120px;background:#06c;color:#fff">Hello</div>';
  document.body.classList.add('sb-show-main');
}, 50);
</script></body></html>`

func TestBrowserRendererShowsAStoryAndReadsItsDocs(t *testing.T) {
	executable, err := findBrowser(systemLocator())
	if err != nil {
		t.Skip("no browser on this machine: " + err.Error())
	}
	folder := t.TempDir()
	index := `{"v":5,"entries":{"ui-label--basic":{"id":"ui-label--basic","type":"story","name":"Basic","title":"UI/Label"}}}`
	for name, body := range map[string]string{"index.json": index, "iframe.html": fakeStorybookPage} {
		if err := os.WriteFile(filepath.Join(folder, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	opened, err := openStorybook(filepath.Dir(folder), filepath.Base(folder))
	if err != nil {
		t.Fatal(err)
	}
	defer opened.close()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	browser, err := startBrowser(ctx, executable)
	if err != nil {
		t.Skip("the browser could not be started here: " + err.Error())
	}
	defer browser.Close()

	story := Story{ID: "ui-label--basic", Name: "Basic", Title: "UI/Label"}
	result := browser.Render(ctx, opened.base, story)
	if result.Failure != "" {
		t.Fatalf("failed: %s", result.Failure)
	}
	if !bytes.HasPrefix(result.PNG, []byte("\x89PNG")) {
		t.Fatalf("the screenshot is not a PNG (%d bytes)", len(result.PNG))
	}
	if code := storySource(result.Meta.Docs); code != `<Label id="ui-label--basic" />` {
		t.Fatalf("source = %q", code)
	}
	if _, ok := result.Meta.ArgTypes["label"]; !ok {
		t.Fatalf("argTypes = %v", result.Meta.ArgTypes)
	}
}
