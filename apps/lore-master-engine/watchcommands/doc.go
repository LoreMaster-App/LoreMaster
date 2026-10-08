// Package watchcommands is what watching a workspace is made of on the engine's side: the
// watch/route method that maps changed files to the generators and pages they affect, and the
// loop that turns a stream of changes into batches (waiting for quiet, running one batch at a
// time, backing off when a batch fails).
package watchcommands
