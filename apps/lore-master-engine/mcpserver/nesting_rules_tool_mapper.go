package mcpserver

import (
	"fmt"
	"strings"

	"lore-master/libs/markdown-workspace/documenttree"
)

const nestingRulesToolName = "loremaster_nesting_rules"

const nestingRulesToolDescription = "Returns LoreMaster's documentation nesting and titling rules, so a Markdown file you create nests under the right page and is titled the way the sync expects."

// nestingRulesTool is the tool's advertised shape. It takes no arguments.
func nestingRulesTool() toolDescriptor {
	return toolDescriptor{
		Name:        nestingRulesToolName,
		Description: nestingRulesToolDescription,
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

// nestingRulesResult renders the conventions as both human-readable text (every MCP
// client shows it) and structured data. The rules come from documenttree, so a rules
// change is reflected here with no edit to this slice.
func nestingRulesResult() toolCallResult {
	conventions := documenttree.NestingConventions()

	var text strings.Builder
	text.WriteString("LoreMaster places and titles pages by these rules. The nesting rules are tried in order; the first that applies decides the page's parent.\n")
	for _, convention := range conventions {
		fmt.Fprintf(&text, "\n- %s: %s", convention.Key, convention.Summary)
	}

	return toolCallResult{
		Content:           []contentBlock{{Type: "text", Text: text.String()}},
		StructuredContent: map[string]any{"conventions": conventions},
	}
}
