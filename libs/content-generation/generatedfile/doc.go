// Package generatedfile defines what every generator shares: the configured Spec it runs
// from, the pages it returns, and the key that marks a file as generated. It depends on
// nothing, so the runner and each generator can depend on it without depending on each other.
package generatedfile
