// Package commentscanning finds the notes in the comments of one file's text.
package commentscanning

import (
	"strings"

	"lore-master/libs/code-notes/commentsyntax"
	"lore-master/libs/code-notes/notetags"
)

// leadingMarks are what may sit between a comment's marker and its text: blanks, the stars down the side of a
// doc block, and the extra slash or bang of /// and //!.
const leadingMarks = " \t*/!"

// scanner is the state of one file: whether a block comment or a multi-line string is open.
type scanner struct {
	syntax  commentsyntax.Syntax
	matcher notetags.Matcher
	notes   []notetags.Note
	block   *commentsyntax.BlockPair
	quote   byte
}

// ScanText returns the notes in text, in order, for a file that writes comments as syntax says. Tags are only taken from
// comments, never from strings, and only at the start of a comment's text on its line. Path is put on every note.
func ScanText(path string, text string, syntax commentsyntax.Syntax, matcher notetags.Matcher) []notetags.Note {
	s := &scanner{syntax: syntax, matcher: matcher}
	for index, line := range strings.Split(text, "\n") {
		s.line(path, index+1, strings.TrimSuffix(line, "\r"))
	}

	return s.notes
}

// line reads one line from the state the previous one left.
func (s *scanner) line(path string, number int, line string) {
	at := 0
	for at < len(line) {
		switch {
		case s.block != nil:
			end := strings.Index(line[at:], s.block.Close)
			if end < 0 {
				s.note(path, number, line, at, len(line))

				return
			}
			s.note(path, number, line, at, at+end)
			at += end + len(s.block.Close)
			s.block = nil
		case s.quote != 0:
			at = s.skipString(line, at)
		default:
			at = s.next(path, number, line, at)
		}
	}
	// A string other than a backtick one does not run past its line.
	if s.quote != '`' {
		s.quote = 0
	}
}

// next handles what starts at line[at] outside any comment or string, and returns where reading goes on.
func (s *scanner) next(path string, number int, line string, at int) int {
	rest := line[at:]
	for index := range s.syntax.Block {
		pair := &s.syntax.Block[index]
		if strings.HasPrefix(rest, pair.Open) {
			s.block = pair

			return at + len(pair.Open)
		}
	}
	for _, marker := range s.syntax.Line {
		if strings.HasPrefix(rest, marker) {
			s.note(path, number, line, at+len(marker), len(line))

			return len(line)
		}
	}
	if strings.IndexByte(s.syntax.Quotes, line[at]) >= 0 {
		s.quote = line[at]

		return at + 1
	}

	return at + 1
}

// skipString moves past the part of an open string on this line, closing it at its quote; a backslash escapes the next character.
func (s *scanner) skipString(line string, at int) int {
	for at < len(line) {
		switch line[at] {
		case '\\':
			at += 2

			continue
		case s.quote:
			s.quote = 0

			return at + 1
		}
		at++
	}

	return len(line)
}

// note looks for a note in the comment text line[from:to], and keeps it with where the tag starts.
func (s *scanner) note(path string, number int, line string, from int, to int) {
	if to > len(line) {
		to = len(line)
	}
	if from >= to {
		return
	}
	text := line[from:to]
	trimmed := strings.TrimLeft(text, leadingMarks)
	found, ok := s.matcher.Parse(strings.TrimRight(trimmed, " \t"))
	if !ok {
		return
	}
	found.Path, found.Line, found.Column = path, number, from+(len(text)-len(trimmed))+1
	s.notes = append(s.notes, found)
}
