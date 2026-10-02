// Package doctext turns source comments into reader-facing text: it drops
// maintainer comments, strips citations a reader cannot resolve, splits a
// member's doc comment into its summary and notes, and wraps text.
package doctext

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"cuelang.org/go/cue/ast"
)

// A citation: "0010:D28", "0010 D28", OQ numbers, ":R2" and "/R1/R2"
// requirement suffixes and "/D9" continuations.
const cite = `\d{4}[: ](?:D|OQ)\d+(?::R\d+(?:/R\d+)*)?(?:/(?:D|OQ)?\d+(?::R\d+(?:/R\d+)*)?)*`

var (
	citeList = cite + `(?:\s*(?:[,;]|and)\s*(?:` + cite + `|D\d+))*`
	// A parenthetical that holds nothing but citations, with an optional
	// lead-in word.
	reCiteParen = regexp.MustCompile(`\s*\((?:(?:see|per|enhancement)\s+)?` + citeList + `\)`)
	// Citations inside running text, with the comma or lead-in before them.
	reCiteInline = regexp.MustCompile(`(?:[,;]\s*)?(?:(?:per|enhancement)\s+)?` + citeList)
	// "See SPEC.md § 3.2." as a sentence, and "SPEC.md § 2.2" inside one.
	reSpecSentence = regexp.MustCompile(`\s*See SPEC\.md\s*§\s*\d+(?:\.\d+)*\.?`)
	reSpecInline   = regexp.MustCompile(`\s*(?:and\s+)?SPEC\.md\s*§\s*\d+(?:\.\d+)*`)
	// Experiment references.
	reExperiment = regexp.MustCompile(`\s*\(?(?:enhancement\s+)?\d{4}\s+experiment\s+\d+\)?|\s*enhancements/\d{4}/experiments/\S+`)
	reAnyRef     = regexp.MustCompile(cite + `|SPEC\.md|\d{4}\s+experiment\s+\d+|enhancements/\d{4}/experiments/`)
	// Clean-up after a removal.
	reEmptyParen  = regexp.MustCompile(`\(\s*[,;]?\s*\)`)
	reSpaceBefore = regexp.MustCompile(`\s+([,.;:)])`)
	reSpaceAfter  = regexp.MustCompile(`\(\s+`)
	reCommaParen  = regexp.MustCompile(`[,;]\s*\)`)
	reParenComma  = regexp.MustCompile(`\(\s*[,;]\s*`)
	reSeeNothing  = regexp.MustCompile(`\bSee\s*\.`)
	reDoubleDot   = regexp.MustCompile(`([^.])\.\s+\.(\s|$)`)
	reSpaces      = regexp.MustCompile(`[ \t]{2,}`)
	reCommentLine = regexp.MustCompile(`^(\s*)//(.*)$`)
	reParaBreak   = regexp.MustCompile(`\n[ \t]*\n`)
)

// Clean removes, from one paragraph of prose (lines joined with single
// spaces), the references a reader cannot resolve: enhancement decision
// citations, SPEC.md section pointers and experiment references. Text with
// none of them is returned unchanged.
func Clean(s string) string {
	if !reAnyRef.MatchString(s) {
		return s
	}
	s = reSpecSentence.ReplaceAllString(s, "")
	s = reSpecInline.ReplaceAllString(s, "")
	s = reExperiment.ReplaceAllString(s, "")
	s = reCiteParen.ReplaceAllString(s, "")
	s = reCiteInline.ReplaceAllString(s, "")
	s = reEmptyParen.ReplaceAllString(s, "")
	s = reCommaParen.ReplaceAllString(s, ")")
	s = reParenComma.ReplaceAllString(s, "(")
	s = reSpaceAfter.ReplaceAllString(s, "(")
	s = reSpaceBefore.ReplaceAllString(s, "$1")
	s = reSeeNothing.ReplaceAllString(s, "")
	s = reDoubleDot.ReplaceAllString(s, "$1.$2")
	s = reSpaces.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// IsWhyLine reports a rationale line, which never reaches a reader.
func IsWhyLine(l string) bool {
	return strings.HasPrefix(strings.TrimSpace(l), "WHY")
}

// StripMaintainerComments removes, everywhere inside n, the comment groups
// that are rationale for maintainers rather than contract: a group whose
// first line starts with WHY, and a //// banner.
func StripMaintainerComments(n ast.Node) {
	ast.Walk(n, func(n ast.Node) bool {
		cgs := ast.Comments(n)
		if len(cgs) == 0 {
			return true
		}
		kept := cgs[:0:0]
		for _, cg := range cgs {
			if len(cg.List) > 0 {
				first := strings.TrimSpace(strings.TrimPrefix(cg.List[0].Text, "//"))
				if strings.HasPrefix(first, "WHY") || strings.HasPrefix(first, "//") {
					continue
				}
			}
			kept = append(kept, cg)
		}
		if len(kept) != len(cgs) {
			ast.SetComments(n, kept)
		}
		return true
	}, nil)
}

// DocText returns a declaration's doc comment, the `//` block directly
// above it, with the comment markers and every WHY line removed and the
// paragraph breaks kept.
func DocText(f *ast.Field) string {
	for _, cg := range ast.Comments(f) {
		if cg.Doc {
			return dropWhyLines(cg.Text())
		}
	}
	return ""
}

// GroupsText joins comment groups (a value's doc) as DocText does.
func GroupsText(groups []*ast.CommentGroup) string {
	var parts []string
	for _, cg := range groups {
		if len(cg.List) > 0 {
			first := strings.TrimSpace(strings.TrimPrefix(cg.List[0].Text, "//"))
			if strings.HasPrefix(first, "WHY") || strings.HasPrefix(first, "//") {
				continue
			}
		}
		if t := dropWhyLines(cg.Text()); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "\n\n")
}

func dropWhyLines(s string) string {
	lines := strings.Split(s, "\n")
	kept := lines[:0]
	for _, l := range lines {
		if !IsWhyLine(l) {
			kept = append(kept, l)
		}
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

// Paragraphs splits text at blank lines.
func Paragraphs(s string) []string {
	var out []string
	for _, p := range reParaBreak.Split(strings.TrimSpace(s), -1) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// CleanParagraphs joins each paragraph's lines with single spaces, cleans
// it, and drops a paragraph the cleaning empties.
func CleanParagraphs(paras []string) []string {
	var out []string
	for _, p := range paras {
		if c := Clean(strings.Join(strings.Fields(p), " ")); c != "" {
			out = append(out, c)
		}
	}
	return out
}

// SplitDoc checks that a member's doc comment opens with its description
// and a period, and returns the remaining paragraphs.
func SplitDoc(doc, desc string) ([]string, error) {
	want := strings.Join(strings.Fields(desc), " ") + "."
	if doc == "" {
		return nil, fmt.Errorf("no doc comment: it must open with the description %q", want)
	}
	paras := Paragraphs(doc)
	first := strings.Join(strings.Fields(paras[0]), " ")
	if !strings.HasPrefix(first, want) {
		return nil, fmt.Errorf("doc comment must open with the description %q, but opens %q", want, truncate(first, len(want)+20))
	}
	rest := strings.TrimSpace(first[len(want):])
	var out []string
	if rest != "" {
		out = append(out, rest)
	}
	return append(out, paras[1:]...), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// CleanCode cleans the comments of a formatted CUE block. A comment
// paragraph (a run of `//` lines at one indent, up to a blank `//` line)
// the rules leave unchanged keeps its line breaks; a changed one is
// re-wrapped to 80 columns less its indent and the marker, at least 40. A
// trailing comment after code is cleaned in place and removed when empty.
func CleanCode(code string) string {
	var out []string
	var run []string // texts of the current comment paragraph
	indent := ""
	flush := func() {
		out = append(out, flushRun(indent, run)...)
		run = nil
	}
	for _, l := range strings.Split(code, "\n") {
		m := reCommentLine.FindStringSubmatch(l)
		if m == nil {
			flush()
			out = append(out, cleanTrailing(l))
			continue
		}
		text := strings.TrimPrefix(m[2], " ")
		if m[1] != indent || strings.TrimSpace(text) == "" {
			flush()
		}
		indent = m[1]
		if strings.TrimSpace(text) == "" {
			out = append(out, commentLine(indent, ""))
			continue
		}
		run = append(run, text)
	}
	flush()
	return strings.Join(out, "\n")
}

func flushRun(indent string, run []string) []string {
	if len(run) == 0 {
		return nil
	}
	var kept []string
	for _, t := range run {
		if !IsWhyLine(t) {
			kept = append(kept, t)
		}
	}
	joined := strings.Join(trimAll(run), " ")
	c := Clean(strings.Join(trimAll(kept), " "))
	var out []string
	switch {
	case c == joined:
		for _, t := range run {
			out = append(out, commentLine(indent, t))
		}
	case c != "":
		width := 80 - utf8.RuneCountInString(strings.ReplaceAll(indent, "\t", "    ")) - 3
		if width < 40 {
			width = 40
		}
		for _, w := range Wrap(c, width) {
			out = append(out, commentLine(indent, w))
		}
	}
	return out
}

func cleanTrailing(l string) string {
	i := strings.Index(l, " // ")
	if i < 0 || !reAnyRef.MatchString(l[i:]) {
		return l
	}
	if c := Clean(l[i+4:]); c != "" {
		return l[:i] + " // " + c
	}
	return l[:i]
}

func commentLine(indent, text string) string {
	if text == "" {
		return indent + "//"
	}
	return indent + "// " + text
}

func trimAll(ls []string) []string {
	out := make([]string, len(ls))
	for i, l := range ls {
		out[i] = strings.TrimSpace(l)
	}
	return out
}

// Wrap breaks s into lines of at most width runes, at spaces.
func Wrap(s string, width int) []string {
	var out []string
	line := ""
	for _, w := range strings.Fields(s) {
		if line != "" && utf8.RuneCountInString(line)+1+utf8.RuneCountInString(w) > width {
			out = append(out, line)
			line = w
			continue
		}
		if line == "" {
			line = w
		} else {
			line += " " + w
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}
