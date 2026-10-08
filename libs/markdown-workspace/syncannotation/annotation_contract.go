package syncannotation

import "time"

// The annotation is one HTML comment block, invisible in every Markdown renderer:
//
//	<!-- lore-master
//	platform: confluence
//	base-url: https://acme.atlassian.net/wiki
//	space: ENG
//	page-id: 123456
//	parent-id: 123000
//	version: 7
//	content-hash: sha256:<64 hex digits>
//	attachments: {"diagram-1.svg":"sha256:<64 hex digits>"}
//	synced-at: 2026-10-01T12:00:00Z
//	title: Optional override of the H1 (the prefix is still applied)
//	parent: ./optional/explicit/parent.md
//	-->
//
// It is the first thing in the file, after an optional byte-order mark, unless the file
// opens with YAML front matter: front matter must stay on line one to be recognised, so
// the block then follows its closing fence. Each line is "key: value". Keys are written
// in the order above, empty ones omitted, unknown ones kept after them in their
// original order. A value never spans lines and never contains "-->".
const (
	OpeningLine = "<!-- lore-master"
	ClosingLine = "-->"
)

// Keys of the contract, in the order they are written.
const (
	KeyPlatform    = "platform"
	KeyBaseURL     = "base-url"
	KeySpace       = "space"
	KeyPageID      = "page-id"
	KeyParentID    = "parent-id"
	KeyVersion     = "version"
	KeyContentHash = "content-hash"
	KeyRenderHash  = "render-hash"
	KeyAttachments = "attachments"
	KeySyncedAt    = "synced-at"
	KeyTitle       = "title"
	KeyParent      = "parent"
	KeyGenerated   = "generated"
)

var knownKeys = []string{
	KeyPlatform, KeyBaseURL, KeySpace, KeyPageID, KeyParentID, KeyVersion,
	KeyContentHash, KeyRenderHash, KeyAttachments, KeySyncedAt, KeyTitle, KeyParent, KeyGenerated,
}

// Annotation is what a file remembers about its page. The sync writes everything but
// Title and Parent, which belong to the author.
type Annotation struct {
	Platform    string
	BaseURL     string
	Space       string
	PageID      string
	ParentID    string
	Version     int
	ContentHash string
	// RenderHash is the hash of the page body as it was converted: it changes when what
	// the page shows changes without the file changing, such as a link to a file that
	// now exists or whose title changed.
	RenderHash string
	// Attachments maps an attachment file name to the content hash it was uploaded with.
	Attachments map[string]string
	SyncedAt    time.Time
	// Title overrides the document's H1 as the page title.
	Title string
	// Parent is an explicit parent document, relative to this file's directory.
	Parent string
	// Generated names the generator that wrote the file (for example "test-results"); a
	// generator only ever overwrites or removes files that carry its own name. The sync
	// keeps it, like Title and Parent.
	Generated string
	// Unknown holds keys this version does not know, so writing never loses them.
	Unknown []Field
}

// Field is one key and its raw value.
type Field struct {
	Key   string
	Value string
}

// Layout is how the file is laid out on disk, so writing the block back changes
// nothing else: its byte-order mark and its line-ending style.
type Layout struct {
	ByteOrderMark bool
	// LineEnding is "\r\n" or "\n", taken from the first line ending in the file.
	LineEnding string
}

// Document is a Markdown file split around its annotation.
type Document struct {
	// Annotation is nil when the file carries none.
	Annotation *Annotation
	// Body is the file without the byte-order mark and without the annotation block,
	// line endings untouched. Front matter, when present, stays in it.
	Body     []byte
	Layout   Layout
	Warnings []string
}
