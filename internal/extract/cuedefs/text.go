package cuedefs

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/open-platform-model/docs-kit/internal/doctext"
)

// block is one paragraph or one verbatim line of a comment.
type block struct {
	text     string
	verbatim bool // a list item or an indented line: kept on its own line
}

// commentLines strips the comment markers from raw "//" lines.
func commentLines(raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, l := range raw {
		l = strings.TrimPrefix(l, "//")
		l = strings.TrimPrefix(l, " ")
		out = append(out, strings.TrimRight(l, " \t"))
	}
	return out
}

// paragraphs splits comment lines into blocks. A blank line ends a
// paragraph; a list item ("- ") or an indented line stands alone. WHY lines
// are dropped. A nil entry in the result marks a paragraph break.
func paragraphs(lines []string) []*block {
	var out []*block
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			out = append(out, &block{text: strings.Join(cur, " ")})
			cur = nil
		}
	}
	for _, l := range lines {
		switch {
		case doctext.IsWhyLine(l):
			continue
		case strings.TrimSpace(l) == "":
			flush()
			out = append(out, nil)
		case strings.HasPrefix(l, "- ") || strings.HasPrefix(l, "  "):
			flush()
			out = append(out, &block{text: l, verbatim: true})
		default:
			cur = append(cur, strings.TrimSpace(l))
		}
	}
	flush()
	return out
}

// cleanComment cleans raw comment lines of a spec block and returns the
// resulting lines, without markers. A paragraph the rules leave unchanged
// keeps its line breaks; a changed one is re-wrapped to width. A group
// whose first line opens with WHY is rationale as a whole and yields
// nothing, and so does a //// banner.
func cleanComment(raw []string, width int) []string {
	lines := commentLines(raw)
	if rationale(lines) {
		return nil
	}
	var out []string
	var para []string
	emit := func(orig []string) {
		if len(orig) == 0 {
			return
		}
		joined := strings.Join(trimAll(orig), " ")
		switch c := doctext.Clean(joined); c {
		case "":
		case joined:
			out = append(out, orig...)
		default:
			out = append(out, doctext.Wrap(c, width)...)
		}
	}
	for _, l := range lines {
		switch {
		case doctext.IsWhyLine(l):
		case strings.TrimSpace(l) == "":
			emit(para)
			para = para[:0]
			if len(out) > 0 && out[len(out)-1] != "" {
				out = append(out, "")
			}
		case strings.HasPrefix(l, "- ") || strings.HasPrefix(l, "  "):
			emit(para)
			para = para[:0]
			if c := doctext.Clean(l); c != "" {
				out = append(out, leadingSpace(l)+c)
			}
		default:
			para = append(para, l)
		}
	}
	emit(para)
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

// rationale reports a group whose first non-blank line opens with WHY or
// is a //// banner.
func rationale(lines []string) bool {
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" {
			return doctext.IsWhyLine(t) || strings.HasPrefix(t, "//")
		}
	}
	return false
}

func trimAll(ls []string) []string {
	out := make([]string, len(ls))
	for i, l := range ls {
		out[i] = strings.TrimSpace(l)
	}
	return out
}

func leadingSpace(s string) string {
	return s[:len(s)-len(strings.TrimLeft(s, " "))]
}

// docProse is a definition's doc comment as public prose: the summary (its
// first sentence), the remaining blocks, and any Example or Usage values.
type docProse struct {
	summary  string
	notes    []*block // nil entries are paragraph breaks
	examples []example
}

type example struct {
	usage bool   // a "Usage:" expression rather than "Example:" values
	text  string // the raw text after the marker
}

var reExampleMarker = regexp.MustCompile(`\b(Example|Usage):\s*`)

// parseDoc turns a definition's raw doc comment into public prose,
// cleaning citations under the policy.
func parseDoc(name string, raw []string, policy doctext.Policy) docProse {
	var d docProse
	var blocks []*block
	for _, b := range paragraphs(commentLines(raw)) {
		if b == nil {
			blocks = append(blocks, nil)
			continue
		}
		t := strings.TrimSpace(policy.Clean(b.text))
		if t == "" {
			continue
		}
		// Example and Usage run to the end of their paragraph.
		if !b.verbatim {
			var ex []example
			t, ex = splitExamples(t)
			d.examples = append(d.examples, ex...)
			if t == "" {
				continue
			}
		}
		blocks = append(blocks, &block{text: t, verbatim: b.verbatim})
	}
	tidy := tidyBreaks(blocks)
	if len(tidy) == 0 {
		return d
	}
	first := stripNameLabel(name, tidy[0].text)
	sum, rest := firstSentence(first)
	d.summary = sum
	if rest != "" {
		tidy[0] = &block{text: rest}
		d.notes = tidy
	} else {
		d.notes = tidy[1:]
		for len(d.notes) > 0 && d.notes[0] == nil {
			d.notes = d.notes[1:]
		}
	}
	return d
}

// splitExamples takes the "Example:" and "Usage:" values off the end of a
// paragraph.
func splitExamples(t string) (string, []example) {
	loc := reExampleMarker.FindStringIndex(t)
	if loc == nil {
		return t, nil
	}
	rest := t[loc[0]:]
	parts := reExampleMarker.FindAllStringSubmatchIndex(rest, -1)
	out := make([]example, 0, len(parts))
	for i, p := range parts {
		end := len(rest)
		if i+1 < len(parts) {
			end = parts[i+1][0]
		}
		out = append(out, example{
			usage: rest[p[2]:p[3]] == "Usage",
			text:  strings.TrimSpace(rest[p[1]:end]),
		})
	}
	return strings.TrimSpace(t[:loc[0]]), out
}

// tidyBreaks drops leading, trailing and doubled paragraph breaks.
func tidyBreaks(blocks []*block) []*block {
	var tidy []*block
	for _, b := range blocks {
		if b == nil && (len(tidy) == 0 || tidy[len(tidy)-1] == nil) {
			continue
		}
		tidy = append(tidy, b)
	}
	for len(tidy) > 0 && tidy[len(tidy)-1] == nil {
		tidy = tidy[:len(tidy)-1]
	}
	return tidy
}

// noteStrings writes note blocks in the model's form: a paragraph as its
// text, a list item as "- item", any other verbatim line indented by two
// spaces, a break as "". A paragraph never starts with "- " or a space:
// such a source line is verbatim.
func noteStrings(blocks []*block) []string {
	out := make([]string, 0, len(blocks))
	for _, b := range blocks {
		switch {
		case b == nil:
			out = append(out, "")
		case b.verbatim && !strings.HasPrefix(b.text, "- "):
			out = append(out, "  "+b.text)
		default:
			out = append(out, b.text)
		}
	}
	return out
}

// stripNameLabel removes a leading "#Name:" or "Name:" label and gives a
// bare leading "Name" its "#", then capitalises the first letter.
func stripNameLabel(name, s string) string {
	bare := strings.TrimPrefix(name, "#")
	for _, p := range []string{name + ":", bare + ":"} {
		if strings.HasPrefix(s, p) {
			return upperFirst(strings.TrimSpace(s[len(p):]))
		}
	}
	if strings.HasPrefix(s, bare+" ") {
		return name + s[len(bare):]
	}
	return upperFirst(s)
}

// upperFirst capitalises a sentence whose first word is a plain word; an
// identifier such as snake_case keeps its spelling.
func upperFirst(s string) string {
	word, _, _ := strings.Cut(s, " ")
	for _, r := range word {
		if !unicode.IsLetter(r) && r != '-' && r != '\'' {
			return s
		}
	}
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || !unicode.IsLower(r) {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}

// firstSentence splits s after its first sentence. A sentence ends at ".",
// "!" or "?" followed by a space; "e.g.", "i.e.", "etc." and "vs." never
// end one.
func firstSentence(s string) (summary, rest string) {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '.' && c != '!' && c != '?' {
			continue
		}
		if i+1 == len(s) {
			return s, ""
		}
		if s[i+1] != ' ' || i+2 >= len(s) {
			continue
		}
		if c == '.' && abbreviation(s[:i+1]) {
			continue
		}
		return s[:i+1], strings.TrimSpace(s[i+2:])
	}
	if !strings.HasSuffix(s, ".") {
		s += "."
	}
	return s, ""
}

func abbreviation(s string) bool {
	for _, a := range []string{"e.g.", "i.e.", "etc.", "vs."} {
		if strings.HasSuffix(s, " "+a) || s == a || strings.HasSuffix(s, "("+a) {
			return true
		}
	}
	return false
}

var reQuoted = regexp.MustCompile(`"(?:[^"\\]|\\.)*"`)

// exampleLines turns "Example:" values into one quoted value per line, and
// a "Usage:" line "expr => result" into the expression with its result as
// a comment.
func exampleLines(ex example) []string {
	if ex.usage {
		if i := strings.Index(ex.text, "=>"); i > 0 {
			return []string{strings.TrimSpace(ex.text[:i]) + " // " + strings.TrimSpace(ex.text[i+2:])}
		}
		return []string{ex.text}
	}
	if vals := reQuoted.FindAllString(ex.text, -1); len(vals) > 0 {
		return vals
	}
	return []string{ex.text}
}
