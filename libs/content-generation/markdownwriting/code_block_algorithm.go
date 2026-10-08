package markdownwriting

import "strings"

// CodeBlock is text in a fenced code block of the given language ("" for none), ending in a
// newline. The fence is longer than any run of backticks in the text, so the text can never
// close the block early.
func CodeBlock(language string, text string) string {
	longest, run := 0, 0
	for _, r := range text {
		if r == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	fence := strings.Repeat("`", max(3, longest+1))

	return fence + language + "\n" + strings.TrimRight(text, "\n") + "\n" + fence + "\n"
}
