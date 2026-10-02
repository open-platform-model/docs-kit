// Package mdtext writes text taken from source into Markdown the site
// renders as written: escaped prose, code spans, table cells and YAML
// front-matter strings.
package mdtext

import (
	"fmt"
	"regexp"
	"strings"
)

// escaper escapes what Markdown or Hugo would read as markup in prose: the
// site renders raw HTML, so `<name>` would vanish, and `{{<` would open a
// shortcode.
var escaper = strings.NewReplacer(
	`\`, `\\`, `<`, `\<`, `>`, `\>`, `*`, `\*`, `_`, `\_`,
	`[`, `\[`, `]`, `\]`, `|`, `\|`, "{{", `{\{`,
)

// Text escapes prose for Markdown, leaving `code spans` as written.
func Text(s string) string {
	var b strings.Builder
	for i, part := range strings.Split(s, "`") {
		if i%2 == 1 {
			b.WriteString("`" + part + "`")
			continue
		}
		b.WriteString(escaper.Replace(part))
	}
	return b.String()
}

// Cell prepares text for a table cell: a pipe ends the cell even inside a
// code span, so every pipe is escaped.
func Cell(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}

// Code wraps s in a code span long enough to hold any backticks in it.
func Code(s string) string {
	fence := "`"
	for strings.Contains(s, fence) {
		fence += "`"
	}
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		return fence + " " + s + " " + fence
	}
	return fence + s + fence
}

// YAMLString quotes a front-matter value as a YAML double-quoted scalar.
func YAMLString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

var reDocNote = regexp.MustCompile(`\bdocs/[a-z0-9-]+\.md\b`)

// DocNotes lists the `docs/<name>.md` names text mentions outside code
// spans, in order of first mention.
func DocNotes(s string) []string {
	var out []string
	seen := map[string]bool{}
	parts := strings.Split(s, "`")
	for i := 0; i < len(parts); i += 2 {
		for _, n := range reDocNote.FindAllString(parts[i], -1) {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	return out
}

// LinkDocNotes links each `docs/<name>.md` named outside a code span in
// escaped text to base+name, when exists reports the note. Note names are
// kebab-case, so escaping leaves them intact.
func LinkDocNotes(s, base string, exists func(string) bool) string {
	parts := strings.Split(s, "`")
	for i := 0; i < len(parts); i += 2 {
		parts[i] = reDocNote.ReplaceAllStringFunc(parts[i], func(n string) string {
			if !exists(n) {
				return n
			}
			return fmt.Sprintf("[%s](%s%s)", n, base, n)
		})
	}
	return strings.Join(parts, "`")
}

// CheckShortcodes refuses a rendered page that still opens a Hugo
// shortcode, `{{<` or `{{%`.
func CheckShortcodes(name, page string) error {
	for i, l := range strings.Split(page, "\n") {
		if strings.Contains(l, "{{<") || strings.Contains(l, "{{%") {
			return fmt.Errorf("%s:%d: the page opens a Hugo shortcode (%q); escape it as {\\{ in the source text", name, i+1, strings.TrimSpace(l))
		}
	}
	return nil
}
