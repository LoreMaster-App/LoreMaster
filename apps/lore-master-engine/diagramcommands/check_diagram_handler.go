package diagramcommands

import (
	"context"
	"strings"

	"lore-master/apps/lore-master-engine/mermaidrendering"
	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/d2rendering"
	"lore-master/libs/documentation-sync/platformport"
)

// MermaidChecker checks Mermaid source; the engine's browser renderer is one.
type MermaidChecker interface {
	Check(ctx context.Context, source string) ([]platformport.DiagramProblem, error)
	Close()
}

// CheckDiagram handles diagram/check: what is wrong with each diagram's source. The browser for
// Mermaid is only started when a Mermaid diagram is asked about, and stopped when the call ends.
func CheckDiagram() rpcserver.Method {
	return CheckDiagramWith(func() MermaidChecker { return mermaidrendering.New() })
}

// CheckDiagramWith is CheckDiagram with the Mermaid checker made by newChecker.
func CheckDiagramWith(newChecker func() MermaidChecker) rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.DiagramCheckParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		var mermaid MermaidChecker
		defer func() {
			if mermaid != nil {
				mermaid.Close()
			}
		}()
		results := make([]rpcprotocol.DiagramCheckOutcome, len(params.Diagrams))
		for i, diagram := range params.Diagrams {
			results[i] = rpcprotocol.DiagramCheckOutcome{Problems: []rpcprotocol.DiagramProblem{}}
			switch strings.ToLower(diagram.Language) {
			case d2rendering.Language:
				results[i].Problems = wire(d2rendering.CheckSyntax(diagram.Source))
			case mermaidrendering.Language:
				if mermaid == nil {
					mermaid = newChecker()
				}
				problems, err := mermaid.Check(ctx, diagram.Source)
				if err != nil {
					// No browser, or Mermaid did not answer: the diagram is not known to be wrong.
					results[i].Unavailable = err.Error()
				} else {
					results[i].Problems = wire(problems)
				}
			}
		}

		return rpcprotocol.DiagramCheckResult{Results: results}, nil
	}
}

func wire(problems []platformport.DiagramProblem) []rpcprotocol.DiagramProblem {
	out := make([]rpcprotocol.DiagramProblem, len(problems))
	for i, problem := range problems {
		out[i] = rpcprotocol.DiagramProblem{Line: problem.Line, Column: problem.Column, Message: problem.Message}
	}

	return out
}
