package mermaidrendering

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"lore-master/libs/documentation-sync/platformport"
)

var parseErrorLine = regexp.MustCompile(`(?i)\bline (\d+)`)

// Check asks Mermaid whether source parses and reports what it refuses, with the line it names
// when it names one. It does not draw. The browser starts on the first call, so a machine with
// none says platformport.ErrRendererUnavailable, which is not a problem with the diagram.
func (r *Renderer) Check(ctx context.Context, source string) ([]platformport.DiagramProblem, error) {
	if strings.TrimSpace(source) == "" {
		return nil, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	tab, err := r.ensureTab(ctx)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(source)
	if err != nil {
		return nil, err
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

	script := `(async () => { try { await mermaid.parse(` + string(encoded) + `); return '' } catch (error) { return String(error && error.message ? error.message : error) } })()`
	var message string
	err = chromedp.Run(timed, chromedp.Evaluate(script, &message, func(params *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams {
		return params.WithAwaitPromise(true)
	}))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			r.dropTab()

			return nil, fmt.Errorf("Mermaid did not answer within %s", r.timeout)
		}

		return nil, err
	}
	if message == "" {
		return nil, nil
	}

	return []platformport.DiagramProblem{{Line: lineOf(message), Message: strings.TrimSpace(message)}}, nil
}

// lineOf is the line Mermaid's message names ("Parse error on line 3"), 0 when it names none.
func lineOf(message string) int {
	match := parseErrorLine.FindStringSubmatch(message)
	if match == nil {
		return 0
	}
	line, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}

	return line
}
