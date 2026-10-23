// Package notescanning finds the notes of a whole workspace: which files to read, and reading them.
package notescanning

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"

	"lore-master/libs/code-notes/commentscanning"
	"lore-master/libs/code-notes/commentsyntax"
	"lore-master/libs/code-notes/notetags"
	"lore-master/libs/markdown-workspace/documentdiscovery"
)

// DefaultMaxFileBytes is the biggest file read: a bigger one is generated or data, not a source file somebody comments.
const DefaultMaxFileBytes = 1 << 20

// Options says what to scan.
type Options struct {
	// WorkspaceRoot is the directory the editor has open, as an OS path.
	WorkspaceRoot string
	// Tags are the tags to find; empty means notetags.DefaultTags.
	Tags []string
	// Excludes are gitignore-syntax patterns (the workspace's ignore list); IncludeGitignored also reads
	// what the .gitignore files ignore. The same rules a sync reads files by.
	Excludes          []string
	IncludeGitignored bool
	// MaxFileBytes is the biggest file read; zero means DefaultMaxFileBytes.
	MaxFileBytes int64
}

// Result is what a scan found.
type Result struct {
	// Notes are in path then line order.
	Notes []notetags.Note
	// FilesScanned counts the files read; FilesSkipped the known kinds left unread (too big, or binary).
	FilesScanned int
	FilesSkipped int
}

// Scan reads every source file of the workspace the ignore rules leave in, in parallel, and returns the notes in their comments.
func Scan(ctx context.Context, options Options) (Result, error) {
	files, err := documentdiscovery.DiscoverFiles(ctx, documentdiscovery.Options{
		WorkspaceRoot: options.WorkspaceRoot, Excludes: options.Excludes, IncludeGitignored: options.IncludeGitignored,
	}, func(rel string) bool {
		_, known := commentsyntax.SyntaxFor(rel)

		return known
	})
	if err != nil {
		return Result{}, err
	}
	limit := options.MaxFileBytes
	if limit <= 0 {
		limit = DefaultMaxFileBytes
	}
	tags := options.Tags
	if len(tags) == 0 {
		tags = notetags.DefaultTags
	}
	matcher := notetags.NewMatcher(tags)

	var (
		mutex   sync.Mutex
		result  Result
		pending = make(chan string)
		workers sync.WaitGroup
	)
	for range runtime.NumCPU() {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for rel := range pending {
				notes, scanned := scanFile(options.WorkspaceRoot, rel, limit, matcher)
				mutex.Lock()
				if scanned {
					result.FilesScanned++
				} else {
					result.FilesSkipped++
				}
				result.Notes = append(result.Notes, notes...)
				mutex.Unlock()
			}
		}()
	}
feed:
	for _, rel := range files {
		select {
		case pending <- rel:
		case <-ctx.Done():
			break feed
		}
	}
	close(pending)
	workers.Wait()
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	slices.SortFunc(result.Notes, func(a, b notetags.Note) int {
		if a.Path != b.Path {
			return compare(a.Path, b.Path)
		}

		return a.Line - b.Line
	})

	return result, nil
}

func compare(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}

	return 0
}

// scanFile reads one file; scanned is false when it was left unread (too big, binary, unreadable).
func scanFile(root string, rel string, limit int64, matcher notetags.Matcher) ([]notetags.Note, bool) {
	syntax, known := commentsyntax.SyntaxFor(rel)
	if !known {
		return nil, false
	}
	path := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	if err != nil || info.Size() > limit {
		return nil, false
	}
	content, err := os.ReadFile(path)
	if err != nil || bytes.IndexByte(content[:min(len(content), 8000)], 0) >= 0 {
		return nil, false
	}

	return commentscanning.ScanText(rel, string(content), syntax, matcher), true
}
