package mermaidrendering

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"lore-master/libs/content-generation/storybookdocs"
	"lore-master/libs/documentation-sync/platformport"
)

// Language is the diagram language this renderer draws.
const Language = "mermaid"

// DefaultTimeout bounds one diagram: one that never settles must not stall a sync.
const DefaultTimeout = 20 * time.Second

// Renderer draws Mermaid in a headless browser. It starts the browser on the first diagram and
// keeps it for the rest; Close stops it. Diagrams are drawn one at a time.
type Renderer struct {
	find    func() (string, error)
	timeout time.Duration

	mu      sync.Mutex
	started bool
	failed  error
	browser context.Context
	cancels []context.CancelFunc
	tab     context.Context
}

var _ platformport.DiagramRenderer = (*Renderer)(nil)

// New makes a renderer that finds the browser the way Storybook generation does:
// LORE_MASTER_BROWSER, then PATH, then the usual install folders.
func New() *Renderer {
	return &Renderer{find: storybookdocs.FindBrowser, timeout: DefaultTimeout}
}

// Render implements platformport.DiagramRenderer.
func (r *Renderer) Render(ctx context.Context, language string, source string) ([]byte, error) {
	if !strings.EqualFold(language, Language) {
		return nil, fmt.Errorf("the Mermaid renderer does not draw %q", language)
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	tab, err := r.ensureTab(ctx)
	if err != nil {
		return nil, err
	}
	svg, err := r.draw(ctx, tab, source)
	if err != nil {
		var timedOut bool
		if errors.Is(err, context.DeadlineExceeded) {
			timedOut = true
			err = fmt.Errorf("the diagram did not finish drawing within %s", r.timeout)
		}
		if timedOut || ctx.Err() == nil && isBrowserGone(err) {
			// The page may be stuck inside Mermaid: the next diagram gets a fresh one.
			r.dropTab()
		}

		return nil, err
	}

	return []byte(svg), nil
}

// Close stops the browser.
func (r *Renderer) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dropTab()
	for _, cancel := range r.cancels {
		cancel()
	}
	r.cancels, r.browser, r.started = nil, nil, false
}

func (r *Renderer) draw(ctx context.Context, tab context.Context, source string) (string, error) {
	encoded, err := json.Marshal(source)
	if err != nil {
		return "", err
	}
	timed, cancel := context.WithTimeout(tab, r.timeout)
	defer cancel()
	go func() {
		select {
		case <-ctx.Done():
			cancel()
		case <-timed.Done():
		}
	}()

	script := `(async () => { window.__loreMasterDiagram = (window.__loreMasterDiagram || 0) + 1; ` +
		`const drawn = await mermaid.render('diagram-' + window.__loreMasterDiagram, ` + string(encoded) + `); return drawn.svg })()`
	var svg string
	err = chromedp.Run(timed, chromedp.Evaluate(script, &svg, func(params *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams {
		return params.WithAwaitPromise(true)
	}))
	if err != nil {
		var exception *cdpruntime.ExceptionDetails
		if errors.As(err, &exception) {
			return "", fmt.Errorf("%s", firstLine(exception.Error()))
		}

		return "", err
	}
	if !strings.Contains(svg, "<svg") {
		return "", errors.New("the browser answered with something that is not an SVG")
	}

	return svg, nil
}

// ensureTab starts the browser if needed and returns a page that has Mermaid loaded.
func (r *Renderer) ensureTab(ctx context.Context) (context.Context, error) {
	if r.tab != nil {
		return r.tab, nil
	}
	if err := r.startBrowser(); err != nil {
		return nil, err
	}
	script, err := bundle()
	if err != nil {
		return nil, fmt.Errorf("the embedded Mermaid library is unreadable: %w", err)
	}

	tab, cancelTab := chromedp.NewContext(r.browser)
	// The tab is created by the first run on its own context: created under a timeout's, it
	// would close when that timeout ends.
	if err := chromedp.Run(tab); err != nil {
		cancelTab()

		return nil, fmt.Errorf("the browser could not open a tab: %w", err)
	}
	setup, cancelSetup := context.WithTimeout(tab, r.timeout)
	defer cancelSetup()
	go func() {
		select {
		case <-ctx.Done():
			cancelSetup()
		case <-setup.Done():
		}
	}()
	err = chromedp.Run(setup,
		chromedp.Navigate("about:blank"),
		chromedp.Evaluate(script, nil),
		chromedp.Evaluate(`mermaid.initialize({ startOnLoad: false })`, nil),
	)
	if err != nil {
		cancelTab()

		return nil, fmt.Errorf("the browser could not load Mermaid: %w", err)
	}
	r.tab = tab
	r.cancels = append(r.cancels, cancelTab)

	return tab, nil
}

func (r *Renderer) dropTab() {
	if r.tab == nil {
		return
	}
	// The tab's cancel is the last one registered.
	last := len(r.cancels) - 1
	r.cancels[last]()
	r.cancels = r.cancels[:last]
	r.tab = nil
}

// startBrowser launches the browser once; a failure to find or start it is remembered and
// reported as the renderer being unavailable, not retried for every diagram.
func (r *Renderer) startBrowser() error {
	if r.started {
		return nil
	}
	if r.failed != nil {
		return r.failed
	}
	executable, err := r.find()
	if err != nil {
		r.failed = fmt.Errorf("%w: %v", platformport.ErrRendererUnavailable, err)

		return r.failed
	}
	options := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(executable),
		chromedp.Flag("hide-scrollbars", true),
	)
	// Chromium refuses to start its sandbox as root, which is what CI containers run as.
	if goruntime.GOOS != "windows" && os.Geteuid() == 0 {
		options = append(options, chromedp.NoSandbox)
	}
	allocator, cancelAllocator := chromedp.NewExecAllocator(context.Background(), options...)
	// A newer browser than the protocol library knows sends events it cannot decode; they are
	// not ours to act on.
	quiet := func(string, ...any) {}
	browser, cancelBrowser := chromedp.NewContext(allocator, chromedp.WithErrorf(quiet))
	if err := chromedp.Run(browser); err != nil {
		cancelBrowser()
		cancelAllocator()
		r.failed = fmt.Errorf("%w: starting %s: %v", platformport.ErrRendererUnavailable, executable, err)

		return r.failed
	}
	r.browser = browser
	r.cancels = append(r.cancels, cancelAllocator, cancelBrowser)
	r.started = true

	return nil
}

func isBrowserGone(err error) bool {
	return errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "target closed") || strings.Contains(err.Error(), "websocket")
}

func firstLine(text string) string {
	if cut := strings.IndexByte(text, '\n'); cut >= 0 {
		return strings.TrimSpace(text[:cut])
	}

	return strings.TrimSpace(text)
}
