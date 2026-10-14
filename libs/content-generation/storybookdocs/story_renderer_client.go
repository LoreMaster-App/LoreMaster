package storybookdocs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

const (
	// renderWidth is the viewport every story is shown in; the screenshot is as tall as the
	// story is, up to maxRenderHeight.
	renderWidth     = 900
	maxRenderHeight = 1600
	// storyTimeout bounds one story: a story that never settles must not stall the run.
	storyTimeout = 45 * time.Second
)

// storyMeta is what the story's own module says about it, read from the running Storybook.
type storyMeta struct {
	ArgTypes map[string]map[string]any `json:"argTypes"`
	Docs     map[string]any            `json:"docs"`
}

// renderedStory is one story after the browser showed it. Failure is set when it could not be
// rendered, and then there is no image.
type renderedStory struct {
	Story   Story
	PNG     []byte
	Meta    storyMeta
	Errors  []string
	Failure string
}

// storyRenderer shows a story and reports what it saw. Implementations never return an error:
// a story that fails is a renderedStory with a Failure, so the others still render.
type storyRenderer interface {
	Render(ctx context.Context, base string, story Story) renderedStory
	Close()
}

// browserRenderer renders stories in a real Chromium-based browser over the DevTools protocol.
type browserRenderer struct {
	browser context.Context
	cancel  []context.CancelFunc
	once    sync.Once
}

// startBrowser launches the browser at executable.
func startBrowser(ctx context.Context, executable string) (*browserRenderer, error) {
	options := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(executable),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("force-device-scale-factor", "1"),
		chromedp.Flag("font-render-hinting", "none"),
		chromedp.WindowSize(renderWidth, 900),
	)
	// Chromium refuses to start its sandbox as root, which is what CI containers run as.
	if goruntime.GOOS != "windows" && os.Geteuid() == 0 {
		options = append(options, chromedp.NoSandbox)
	}
	allocator, cancelAllocator := chromedp.NewExecAllocator(ctx, options...)
	// A newer browser than the protocol library knows sends events it cannot decode; they are
	// not ours to act on and would only fill the engine's log.
	quiet := func(string, ...any) {}
	browser, cancelBrowser := chromedp.NewContext(allocator, chromedp.WithErrorf(quiet))
	if err := chromedp.Run(browser); err != nil {
		cancelBrowser()
		cancelAllocator()

		return nil, fmt.Errorf("starting %s: %w", executable, err)
	}

	return &browserRenderer{browser: browser, cancel: []context.CancelFunc{cancelBrowser, cancelAllocator}}, nil
}

// Close stops the browser.
func (r *browserRenderer) Close() {
	r.once.Do(func() {
		for _, cancel := range r.cancel {
			cancel()
		}
	})
}

// Render opens the story in its own tab, waits until it has drawn, and takes the screenshot.
func (r *browserRenderer) Render(ctx context.Context, base string, story Story) renderedStory {
	result := renderedStory{Story: story}
	tab, cancelTab := chromedp.NewContext(r.browser)
	defer cancelTab()
	var errorsMu sync.Mutex
	chromedp.ListenTarget(tab, func(event any) {
		errorsMu.Lock()
		defer errorsMu.Unlock()
		switch e := event.(type) {
		case *cdpruntime.EventExceptionThrown:
			result.Errors = append(result.Errors, e.ExceptionDetails.Error())
		case *cdpruntime.EventConsoleAPICalled:
			if e.Type == cdpruntime.APITypeError && len(e.Args) > 0 {
				result.Errors = append(result.Errors, strings.Trim(string(e.Args[0].Value), `"`))
			}
		}
	})

	timed, cancelTimeout := context.WithTimeout(tab, storyTimeout)
	defer cancelTimeout()
	go func() {
		select {
		case <-ctx.Done():
			cancelTimeout()
		case <-timed.Done():
		}
	}()

	png, meta, err := showStory(timed, base, story.ID)
	errorsMu.Lock()
	defer errorsMu.Unlock()
	if err != nil {
		result.Failure = err.Error()

		return result
	}
	result.PNG, result.Meta = png, meta

	return result
}

// waitForStory is true once Storybook has drawn the story, or shows its error display.
const waitForStory = `(document.body.classList.contains('sb-show-main') && document.querySelector('#storybook-root') && document.querySelector('#storybook-root').children.length > 0) || document.body.classList.contains('sb-show-errordisplay')`

// settle stops animations and waits for fonts, so two renders of the same story match.
const settle = `(async()=>{const st=document.createElement('style');st.textContent='*,*::before,*::after{animation:none!important;transition:none!important;caret-color:transparent!important}';document.head.appendChild(st);await document.fonts.ready;await new Promise(r=>setTimeout(r,300));return document.body.className})()`

// readMeta asks the running Storybook for the story's argTypes and documentation parameters.
const readMeta = `(async(id)=>{try{const p=window.__STORYBOOK_PREVIEW__;const sv=p.storyStoreValue||p.storyStore;const s=await sv.loadStory({storyId:id});const strip=o=>o===undefined?undefined:JSON.parse(JSON.stringify(o,(k,v)=>typeof v==='function'?undefined:v));return JSON.stringify({argTypes:strip(s.argTypes),docs:strip(s.parameters&&s.parameters.docs)})}catch(e){return JSON.stringify({})}})(%q)`

// contentBottom is how far down the story's content reaches.
const contentBottom = `(()=>{let b=0;for(const el of document.body.children){if(['SCRIPT','STYLE','LINK'].includes(el.tagName)||el.id==='storybook-docs')continue;const r=el.getBoundingClientRect();if(r.width*r.height<=0)continue;b=Math.max(b,r.bottom)}return JSON.stringify({bottom:b})})()`

func showStory(ctx context.Context, base string, id string) ([]byte, storyMeta, error) {
	var state, metaJSON, extent string
	awaiting := func(p *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams { return p.WithAwaitPromise(true) }
	err := chromedp.Run(ctx,
		chromedp.Navigate(base+"/iframe.html?viewMode=story&id="+url.QueryEscape(id)),
		chromedp.Poll(waitForStory, nil, chromedp.WithPollingTimeout(30*time.Second)),
		chromedp.Evaluate(settle, &state, awaiting),
		chromedp.Evaluate(fmt.Sprintf(readMeta, id), &metaJSON, awaiting),
		chromedp.Evaluate(contentBottom, &extent),
	)
	if err != nil {
		return nil, storyMeta{}, fmt.Errorf("the story did not render: %w", err)
	}
	if strings.Contains(state, "sb-show-errordisplay") {
		return nil, storyMeta{}, fmt.Errorf("the story threw while rendering")
	}

	var meta storyMeta
	_ = json.Unmarshal([]byte(metaJSON), &meta)
	var measured struct {
		Bottom float64 `json:"bottom"`
	}
	_ = json.Unmarshal([]byte(extent), &measured)
	height := min(max(measured.Bottom+16, 40), maxRenderHeight)

	var png []byte
	err = chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
		data, shotErr := page.CaptureScreenshot().
			WithFormat(page.CaptureScreenshotFormatPng).
			WithCaptureBeyondViewport(true).
			WithClip(&page.Viewport{X: 0, Y: 0, Width: renderWidth, Height: height, Scale: 1}).
			Do(c)
		png = data

		return shotErr
	}))
	if err != nil {
		return nil, storyMeta{}, fmt.Errorf("the screenshot failed: %w", err)
	}

	return png, meta, nil
}
