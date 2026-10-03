package goapi

import (
	"fmt"
	"go/doc/comment"
	"regexp"
	"strconv"
	"strings"

	"github.com/open-platform-model/docs-kit/internal/doctext"
	"github.com/open-platform-model/docs-kit/internal/mdtext"
)

// mdPrinter writes a parsed doc comment as Markdown the site renders as
// written. go/doc/comment's own Markdown printer escapes no shortcode,
// writes heading IDs the page dialect does not know and indents code, so
// this one prints the same tree under docs-kit's rules: prose escaped,
// code spans kept, links checked, code in sized fences.
type mdPrinter struct {
	policy doctext.Policy
	docURL func(*comment.DocLink) string // a doc link's URL
	where  string                        // names the comment in errors: "kernel.New (opm/kernel/kernel.go)"
}

// markdown prints d with its headings at level and below.
func (pr *mdPrinter) markdown(d *comment.Doc, level int) (string, error) {
	var blocks []string
	for _, x := range d.Content {
		s, err := pr.block(x, level)
		if err != nil {
			return "", err
		}
		if s != "" {
			blocks = append(blocks, s)
		}
	}
	return strings.Join(blocks, "\n\n"), nil
}

func (pr *mdPrinter) block(x comment.Block, level int) (string, error) {
	switch x := x.(type) {
	case *comment.Paragraph:
		return pr.text(x.Text, false)
	case *comment.Heading:
		t, err := pr.text(x.Text, true)
		if t == "" || err != nil {
			return "", err
		}
		if level > 6 {
			return "**" + t + "**", nil
		}
		return strings.Repeat("#", level) + " " + t, nil
	case *comment.Code:
		code := strings.TrimRight(x.Text, "\n")
		f := Fence(code)
		return f + "text\n" + code + "\n" + f, nil
	case *comment.List:
		return pr.list(x)
	}
	return "", fmt.Errorf("%s: unknown doc comment block %T", pr.where, x)
}

func (pr *mdPrinter) list(x *comment.List) (string, error) {
	var b strings.Builder
	sep := "\n"
	if x.BlankBetween() {
		sep = "\n\n"
	}
	n := 0
	for _, item := range x.Items {
		prefix := "- "
		if item.Number != "" {
			prefix = item.Number + ". "
		}
		var paras []string
		for _, blk := range item.Content {
			p, ok := blk.(*comment.Paragraph)
			if !ok {
				return "", fmt.Errorf("%s: unknown list item block %T", pr.where, blk)
			}
			t, err := pr.text(p.Text, false)
			if err != nil {
				return "", err
			}
			if t != "" {
				paras = append(paras, t)
			}
		}
		if len(paras) == 0 {
			continue
		}
		if n > 0 {
			b.WriteString(sep)
		}
		n++
		indent := strings.Repeat(" ", len(prefix))
		b.WriteString(prefix + strings.Join(paras, "\n\n"+indent))
	}
	return b.String(), nil
}

// sub is a link that stands in the prose as a placeholder while the
// citation policy runs: its Markdown, and its text as written for a code
// span, where no link renders.
type sub struct{ md, plain string }

func subMark(i int) string { return "\x01" + strconv.Itoa(i) + "\x01" }

var reSubMark = regexp.MustCompile("^\x01(\\d+)\x01")

// text prints one paragraph or heading: its lines joined, the citation
// policy applied, then escaped outside code spans and links.
func (pr *mdPrinter) text(xs []comment.Text, heading bool) (string, error) {
	var raw strings.Builder
	var subs []sub
	for _, x := range xs {
		switch x := x.(type) {
		case comment.Plain:
			raw.WriteString(string(x))
		case comment.Italic:
			raw.WriteString(string(x))
		case *comment.Link:
			if err := checkURL(x.URL); err != nil {
				return "", fmt.Errorf("%s: %w", pr.where, err)
			}
			plain := plainText(x.Text)
			src := "[" + plain + "]" // as written, for a code span
			if x.Auto {
				src = plain
			}
			raw.WriteString(subMark(len(subs)))
			subs = append(subs, sub{md: "[" + escape(plain, heading) + "](" + encodeURL(x.URL) + ")", plain: src})
		case *comment.DocLink:
			plain := plainText(x.Text)
			raw.WriteString(subMark(len(subs)))
			subs = append(subs, sub{md: "[" + escape(plain, heading) + "](" + pr.docURL(x) + ")", plain: "[" + plain + "]"})
		}
	}
	s := strings.Join(strings.Fields(raw.String()), " ")
	s = pr.policy.Clean(s)
	return pr.inline(s, subs, heading), nil
}

func plainText(xs []comment.Text) string {
	var b strings.Builder
	for _, x := range xs {
		switch x := x.(type) {
		case comment.Plain:
			b.WriteString(string(x))
		case comment.Italic:
			b.WriteString(string(x))
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// reDecisionLink matches the link the "link" citation policy writes.
var reDecisionLink = regexp.MustCompile(`^\[\d{4}:D[0-9DR:/]*\]\(/enhancements/\d{4}/decisions/\)`)

// inline escapes cleaned prose outside its code spans (a run of n
// backticks up to the next run of exactly n, rewritten as a span longer
// than any backtick run it holds), its placeholders' links and, under the
// "link" policy, its decision links; then escapes what would open a block
// at the start of the line.
func (pr *mdPrinter) inline(s string, subs []sub, heading bool) string {
	var b strings.Builder
	var text strings.Builder
	flush := func() {
		b.WriteString(escape(text.String(), heading))
		text.Reset()
	}
	for i := 0; i < len(s); {
		if m := reSubMark.FindStringSubmatch(s[i:]); m != nil {
			n, _ := strconv.Atoi(m[1])
			flush()
			b.WriteString(subs[n].md)
			i += len(m[0])
			continue
		}
		if pr.policy == doctext.Link && s[i] == '[' {
			if m := reDecisionLink.FindString(s[i:]); m != "" {
				flush()
				b.WriteString(m)
				i += len(m)
				continue
			}
		}
		if s[i] == '`' {
			n := backtickRun(s, i)
			// A run of three at the start of a line would open a fence.
			if end := closingRun(s, i+n, n); end >= 0 && (i > 0 || n < 3) {
				flush()
				inner := reSubMarkAny.ReplaceAllStringFunc(s[i+n:end], func(mk string) string {
					k, _ := strconv.Atoi(strings.Trim(mk, "\x01"))
					return subs[k].plain
				})
				b.WriteString(mdtext.Code(trimSpan(inner)))
				i = end + n
				continue
			}
			text.WriteString(s[i : i+n])
			i += n
			continue
		}
		text.WriteByte(s[i])
		i++
	}
	flush()
	return lineStart(b.String())
}

var reSubMarkAny = regexp.MustCompile("\x01\\d+\x01")

// trimSpan removes the one space CommonMark strips from each side of a
// code span's content when both are there, so mdtext.Code can add its own.
func trimSpan(s string) string {
	if len(s) >= 2 && s[0] == ' ' && s[len(s)-1] == ' ' && strings.Trim(s, " ") != "" {
		return s[1 : len(s)-1]
	}
	return s
}

// escaped is every character prose escapes with a backslash: what Markdown
// reads as emphasis, a link or an image, a table cell, raw HTML or a code
// span.
const escaped = "\\`*_[]<>|!"

// escape backslash-escapes prose, and writes "{{" as "{\{" until none is
// left, so no text opens a Hugo shortcode. A heading also escapes "#",
// which would close it.
func escape(s string, heading bool) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if strings.IndexByte(escaped, c) >= 0 || (heading && c == '#') {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	out := b.String()
	for strings.Contains(out, "{{") {
		out = strings.ReplaceAll(out, "{{", `{\{`)
	}
	return out
}

var reOrdered = regexp.MustCompile(`^\d+[.)]`)

// lineStart escapes what would open a list, a heading, a quote or a fence
// at the start of a line.
func lineStart(s string) string {
	if s == "" {
		return s
	}
	if strings.IndexByte("#+-=~", s[0]) >= 0 {
		return `\` + s
	}
	if m := reOrdered.FindString(s); m != "" {
		return m[:len(m)-1] + `\` + s[len(m)-1:]
	}
	return s
}

// reScheme matches a URL that names a scheme.
var reScheme = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9+.-]*):`)

// checkURL refuses a link whose URL names a scheme other than http or
// https: the site renders links as written, so a javascript: link would
// run.
func checkURL(u string) error {
	if m := reScheme.FindStringSubmatch(u); len(m) == 2 && !strings.EqualFold(m[1], "http") && !strings.EqualFold(m[1], "https") {
		return fmt.Errorf("the doc comment links %q; a link is http, https or relative", u)
	}
	return nil
}

// urlEscaper percent-encodes what would end a Markdown link destination
// or open markup inside it.
var urlEscaper = strings.NewReplacer(
	" ", "%20", "(", "%28", ")", "%29", "<", "%3C", ">", "%3E",
	"`", "%60", "{", "%7B", "}", "%7D", `\`, "%5C", `"`, "%22",
)

func encodeURL(u string) string { return urlEscaper.Replace(u) }

func backtickRun(s string, i int) int {
	n := 0
	for i+n < len(s) && s[i+n] == '`' {
		n++
	}
	return n
}

// closingRun finds the next run of exactly n backticks from i, or -1.
func closingRun(s string, i, n int) int {
	for i < len(s) {
		if s[i] != '`' {
			i++
			continue
		}
		m := backtickRun(s, i)
		if m == n {
			return i
		}
		i += m
	}
	return -1
}

// Fence is a code fence longer than any backtick run in code, at least
// three backticks.
func Fence(code string) string {
	longest, run := 0, 0
	for _, c := range code {
		if c == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	return strings.Repeat("`", max(3, longest+1))
}
