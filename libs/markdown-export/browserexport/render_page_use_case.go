package browserexport

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Options say how the page is printed.
type Options struct {
	// BrowserPath is the browser to run; empty finds one (see LocateBrowser).
	BrowserPath string
	// PageSize is the PDF's paper, one of PageSizes; A4 when empty or unknown.
	PageSize string
	// Width is the PNG's width in pixels; 1000 when zero.
	Width int
	// MermaidPath is the Mermaid library (mermaid.min.js) diagrams are drawn with; without one, or one that cannot be
	// found, diagrams are printed as code.
	MermaidPath string
}

// DefaultWidth is how wide a PNG is when nothing says otherwise.
const DefaultWidth = 1000

// limit keeps a browser that hangs from hanging the export.
const limit = 90 * time.Second

// RenderPDF prints a complete HTML page to PDF.
func RenderPDF(page string, options Options) ([]byte, error) {
	return withBrowser(options, func(run runner) ([]byte, error) {
		mermaid := LocateMermaid(options.MermaidPath)
		file, err := run.writePage(withDiagrams(withHeadBefore(page, lightTheme+pageStyle(validPageSize(options.PageSize))), mermaid))
		if err != nil {
			return nil, err
		}
		output := filepath.Join(run.folder, "page.pdf")
		if _, err := run.browse(append(waitForDiagrams(page, mermaid), "--no-pdf-header-footer", "--print-to-pdf="+output, file)...); err != nil {
			return nil, err
		}

		return run.result(output)
	})
}

// RenderPNG takes a screenshot of the whole of a complete HTML page, as wide as Options.Width.
func RenderPNG(page string, options Options) ([]byte, error) {
	width := options.Width
	if width <= 0 {
		width = DefaultWidth
	}

	return withBrowser(options, func(run runner) ([]byte, error) {
		mermaid := LocateMermaid(options.MermaidPath)
		file, err := run.writePage(withDiagrams(withHeadBefore(page, lightTheme+heightProbe), mermaid))
		if err != nil {
			return nil, err
		}
		size := fmt.Sprintf("--window-size=%d,%d", width, 800)
		dump, err := run.browse(size, "--virtual-time-budget=15000", "--dump-dom", file)
		if err != nil {
			return nil, err
		}
		height, found := pageHeight(string(dump))
		if !found {
			return nil, fmt.Errorf("could not measure the page with %s", filepath.Base(run.browser))
		}

		output := filepath.Join(run.folder, "page.png")
		shot := append(waitForDiagrams(page, mermaid), fmt.Sprintf("--window-size=%d,%d", width, height), "--hide-scrollbars", "--screenshot="+output, file)
		if _, err := run.browse(shot...); err != nil {
			return nil, err
		}

		return run.result(output)
	})
}

// waitForDiagrams is what the browser is told so that a page that draws diagrams is not printed before they are drawn: it
// lets the page's timers and promises run on a virtual clock first.
func waitForDiagrams(page, mermaidPath string) []string {
	if mermaidPath == "" || !hasDiagrams(page) {
		return nil
	}

	return []string{"--virtual-time-budget=15000"}
}

// runner is one browser run: the browser, and a folder of its own for the page, the browser's profile
// and what it writes.
type runner struct {
	browser string
	folder  string
}

func withBrowser(options Options, work func(runner) ([]byte, error)) ([]byte, error) {
	browser, err := LocateBrowser(options.BrowserPath)
	if err != nil {
		return nil, err
	}
	folder, err := os.MkdirTemp("", "lore-master-export-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(folder) }()

	return work(runner{browser: browser, folder: folder})
}

// writePage stores the page and returns the address the browser opens it at.
func (r runner) writePage(page string) (string, error) {
	path := filepath.Join(r.folder, "page.html")
	if err := os.WriteFile(path, []byte(page), 0o600); err != nil {
		return "", err
	}

	return fileURL(path), nil
}

func fileURL(path string) string {
	slashed := filepath.ToSlash(path)
	if !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed
	}

	return "file://" + slashed
}

// browse runs the browser headless on a page and returns what it printed. A profile folder of its own
// makes it a new browser every time: one that is already open would otherwise take the page and exit.
func (r runner) browse(arguments ...string) ([]byte, error) {
	profile := filepath.Join(r.folder, "profile")
	command := []string{"--headless", "--disable-gpu", "--no-first-run", "--no-default-browser-check", "--user-data-dir=" + profile}
	if runtime.GOOS == "linux" && os.Geteuid() == 0 {
		// A browser refuses to start as root without this, which is how a container runs.
		command = append(command, "--no-sandbox")
	}
	command = append(command, arguments...)

	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	var stdout, stderr bytes.Buffer
	process := exec.CommandContext(ctx, r.browser, command...)
	process.Stdout, process.Stderr = &stdout, &stderr
	if err := process.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%s took longer than %s", filepath.Base(r.browser), limit)
		}

		return nil, fmt.Errorf("%s failed: %w: %s", filepath.Base(r.browser), err, tail(stderr.String()))
	}

	return stdout.Bytes(), nil
}

// result reads the file the browser wrote. Some browsers return before the file is complete, so a missing
// file is waited for briefly before it is reported.
func (r runner) result(path string) ([]byte, error) {
	deadline := time.Now().Add(10 * time.Second)
	for {
		data, err := os.ReadFile(path)
		if err == nil && len(data) > 0 {
			return data, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("%s did not write the file", filepath.Base(r.browser))
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// tail keeps the end of a browser's complaints, which is where the reason is.
func tail(text string) string {
	text = strings.TrimSpace(text)
	if len(text) > 400 {
		return "…" + text[len(text)-400:]
	}

	return text
}
