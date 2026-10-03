package goapi

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Anchor is Hugo's default ("github") heading anchor of a heading's text:
// lower case, every space and "-" as "-", every other character than a
// letter, a digit or "_" dropped. "Kernel.Render" is "kernelrender".
func Anchor(text string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(text) {
		switch {
		case r == ' ' || r == '-':
			b.WriteByte('-')
		case r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// reLinkDest matches the destination of an inline Markdown link, which a
// heading's text does not show.
var reLinkDest = regexp.MustCompile(`\]\([^)]*\)`)

// HeadingAnchor is the anchor of a Markdown heading's text as written: a
// link's destination is not text, and escapes and markup are characters
// Anchor drops.
func HeadingAnchor(md string) string {
	return Anchor(reLinkDest.ReplaceAllString(md, "]"))
}

// Anchors hands out a page's heading anchors in page order, as Hugo makes
// them unique: a repeated anchor gets "-1", then "-2", and an empty one is
// "heading".
type Anchors struct {
	seen map[string]bool
}

// Next records the next heading's anchor and returns it, made unique.
func (a *Anchors) Next(anchor string) string {
	if a.seen == nil {
		a.seen = map[string]bool{}
	}
	if anchor == "" {
		anchor = "heading"
	}
	if !a.seen[anchor] {
		a.seen[anchor] = true
		return anchor
	}
	for i := 1; ; i++ {
		c := anchor + "-" + strconv.Itoa(i)
		if !a.seen[c] {
			a.seen[c] = true
			return c
		}
	}
}

// reHeading matches an ATX heading line.
var reHeading = regexp.MustCompile(`^(#{1,6}) (.*)$`)

// Headings lists the text of every heading line of Markdown, in order,
// skipping fenced code.
func Headings(md string) []string {
	var out []string
	fence := ""
	for _, l := range strings.Split(md, "\n") {
		t := strings.TrimLeft(l, " ")
		if fence != "" {
			if strings.HasPrefix(t, fence) && strings.Trim(t, "`") == "" {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(t, "```") {
			fence = t[:len(t)-len(strings.TrimLeft(t, "`"))]
			continue
		}
		if m := reHeading.FindStringSubmatch(l); m != nil {
			out = append(out, m[2])
		}
	}
	return out
}
