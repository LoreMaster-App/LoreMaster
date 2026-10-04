// Package attachments keeps a page's files (images, rendered diagrams) in sync: it uploads
// them for a push and lists and downloads them for a two-way pull, through REST v1 on every
// edition (Cloud's v2 has no attachment endpoints).
package attachments
