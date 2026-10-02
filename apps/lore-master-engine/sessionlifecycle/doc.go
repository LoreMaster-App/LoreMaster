// Package sessionlifecycle opens and closes connections to documentation platforms. A
// session holds a verified connection, credential included, in memory only: nothing
// about it is written to disk or logged, and closing it drops every reference.
package sessionlifecycle
