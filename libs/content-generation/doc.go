// Package contentgeneration is the content-generation capability: generators that read
// project artifacts (test reports, source documentation, API descriptions) and write them as
// ordinary Markdown files into the workspace. Because the output is plain Markdown, the
// existing discovery, page tree, annotations, links and every storage handle it unchanged,
// and it stays reviewable in git. Its code lives in the slice packages below this
// directory, one per outcome.
package contentgeneration
