package enhancements

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/open-platform-model/docs-kit/internal/mdtext"
)

// clean applies the transforms, in order, to one source file's text:
// refuse a Hugo shortcode delimiter or context marker; remove HTML
// comments; remove the first heading; then, outside code fences, tag an
// untagged opening fence "text", resolve every relative link and read a
// link whose text is its target's file name as the target page's title,
// and escape a "<" that would open raw HTML.
func (x *extraction) clean(file, text string) (string, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	raw := strings.Split(text, "\n")
	for i, l := range raw {
		if strings.Contains(l, "{{<") || strings.Contains(l, "{{%") || strings.Contains(l, "{{__hugo_ctx") {
			return "", fmt.Errorf("%s:%d: holds a Hugo shortcode delimiter ({{< or {{%%) or Hugo's context marker ({{__hugo_ctx), which the site would act on; remove it", file, i+1)
		}
	}
	lines, err := dropComments(file, raw)
	if err != nil {
		return "", err
	}
	lines = dropTitle(lines)
	out := make([]string, len(lines))
	var fence string
	for i, l := range lines {
		out[i] = l.text
		if f, tagged, ok := fenceLine(l.text, fence); ok {
			switch {
			case fence != "":
				fence = ""
			case !tagged:
				fence = f
				out[i] = tagFence(l.text)
			default:
				fence = f
			}
			continue
		}
		if fence != "" {
			continue
		}
		if out[i], err = x.line(file, l.text); err != nil {
			return "", fmt.Errorf("%s:%d: %w", file, l.n, err)
		}
	}
	return strings.Join(out, "\n"), nil
}

// srcLine is a line of text and its line number in the source file.
type srcLine struct {
	n    int
	text string
}

// fenceLine reads a code fence line as the dialect lint does: a run of
// three or more backticks or tildes after at most three spaces. Outside a
// fence (open == "") every such line opens one; inside, only a run of the
// opening character at least as long, with nothing after it but blanks,
// closes it. It returns the run and whether an opening fence names a
// language.
func fenceLine(l, open string) (run string, tagged, ok bool) {
	t := strings.TrimLeft(l, " ")
	if len(l)-len(t) > 3 || !strings.HasPrefix(t, "```") && !strings.HasPrefix(t, "~~~") {
		return "", false, false
	}
	run = t[:len(t)-len(strings.TrimLeft(t, t[:1]))]
	if open == "" {
		return run, strings.TrimLeft(t, "`~ \t") != "", true
	}
	if run[0] == open[0] && len(run) >= len(open) && strings.TrimSpace(t[len(run):]) == "" {
		return run, false, true
	}
	return "", false, false
}

// tagFence adds the language "text" to an untagged opening fence.
func tagFence(l string) string {
	t := strings.TrimLeft(l, " ")
	run := t[:len(t)-len(strings.TrimLeft(t, t[:1]))]
	return l[:len(l)-len(t)] + run + "text"
}

// dropComments removes HTML comments outside code fences and code spans,
// across lines. A line that held nothing but comments is removed whole, so
// a comment inside a table or a list does not split it; an unclosed
// comment is refused.
func dropComments(file string, lines []string) ([]srcLine, error) {
	var out []srcLine
	fence := ""
	open := 0 // the line an unclosed comment opened on, 1-based; 0 outside one
	kept := ""
	for i, l := range lines {
		if open == 0 {
			if f, _, ok := fenceLine(l, fence); ok {
				if fence == "" {
					fence = f
				} else {
					fence = ""
				}
				out = append(out, srcLine{i + 1, l})
				continue
			}
			if fence != "" {
				out = append(out, srcLine{i + 1, l})
				continue
			}
		}
		rest, had := l, false
		var b strings.Builder
		if open > 0 {
			end := strings.Index(rest, "-->")
			if end < 0 {
				continue
			}
			rest, open = rest[end+3:], 0
			b.WriteString(kept)
			had = true
		}
		for {
			at := commentStart(rest)
			if at < 0 {
				b.WriteString(rest)
				break
			}
			had = true
			b.WriteString(rest[:at])
			end := strings.Index(rest[at+4:], "-->")
			if end < 0 {
				open, kept = i+1, b.String()
				break
			}
			rest = rest[at+4+end+3:]
		}
		if open > 0 {
			continue
		}
		s := b.String()
		if had && strings.TrimSpace(s) == "" {
			continue
		}
		out = append(out, srcLine{i + 1, s})
	}
	if open > 0 {
		return nil, fmt.Errorf("%s:%d: an HTML comment opens with <!-- and never closes with -->", file, open)
	}
	return out, nil
}

// commentStart is the index of the first "<!--" in l outside a code span,
// or -1.
func commentStart(l string) int {
	code := codeMask(l)
	for i := strings.Index(l, "<!--"); i >= 0; {
		if !code[i] {
			return i
		}
		j := strings.Index(l[i+1:], "<!--")
		if j < 0 {
			return -1
		}
		i += 1 + j
	}
	return -1
}

// dropTitle removes the leading blank lines and, when the first line left
// is a level-one heading, that line and the blank lines after it: the
// site prints the title from the front matter.
func dropTitle(lines []srcLine) []srcLine {
	lines = trimBlank(lines)
	if len(lines) > 0 && strings.HasPrefix(lines[0].text, "# ") {
		lines = trimBlank(lines[1:])
	}
	return lines
}

func trimBlank(lines []srcLine) []srcLine {
	for len(lines) > 0 && strings.TrimSpace(lines[0].text) == "" {
		lines = lines[1:]
	}
	return lines
}

// codeMask marks the bytes of l inside a code span, delimiters included:
// a run of backticks opens a span closed by the next run of the same
// length on the line; a run with no partner is literal.
func codeMask(l string) []bool {
	mask := make([]bool, len(l))
	for i := 0; i < len(l); {
		if l[i] != '`' || escaped(l, i) {
			i++
			continue
		}
		n := runLen(l, i)
		end := -1
		for j := i + n; j < len(l); {
			if l[j] != '`' {
				j++
				continue
			}
			m := runLen(l, j)
			if m == n {
				end = j + m
				break
			}
			j += m
		}
		if end < 0 {
			i += n
			continue
		}
		for k := i; k < end; k++ {
			mask[k] = true
		}
		i = end
	}
	return mask
}

func runLen(l string, i int) int {
	n := 0
	for i+n < len(l) && l[i+n] == l[i] {
		n++
	}
	return n
}

// reRefDef is a reference definition, "[label]: destination"; a footnote
// ("[^1]:") is not one.
var reRefDef = regexp.MustCompile(`^( {0,3}\[[^\]^][^\]]*\]:[ \t]*)(<[^>]*>|\S+)`)

// line resolves the links of one line outside a code fence and escapes
// the "<" that would open raw HTML, outside code spans.
func (x *extraction) line(file, l string) (string, error) {
	if m := reRefDef.FindStringSubmatchIndex(l); len(m) > 5 && !codeMask(l)[m[4]] {
		return x.refDef(file, l, m[4], m[5])
	}
	var b strings.Builder
	code := codeMask(l)
	last := 0 // l[:last] is in b
	for i := 0; i+1 < len(l); i++ {
		if l[i] != ']' || l[i+1] != '(' || code[i] || escaped(l, i) {
			continue
		}
		ds, de := destination(l, i+2)
		if ds < 0 {
			continue
		}
		angled := ds > i+2 // "](<dest>)": ds is past the "<"
		dest := l[ds:de]
		u, title, err := x.resolve(file, dest)
		if err != nil {
			return "", err
		}
		x.writeText(&b, l, last, i, code, title, dest)
		b.WriteString("](")
		if angled && (u == dest || strings.ContainsAny(u, " \t")) {
			u = "<" + u + ">" // a destination with a blank needs its brackets
		}
		b.WriteString(u)
		last = de
		if angled {
			last++ // past the closing ">"
		}
		i = last - 1
	}
	b.WriteString(escapeHTML(l[last:], 0))
	return b.String(), nil
}

// writeText writes l[last:end], the text before a link's "](" at end:
// a link text that is its target's file name (dest's base) becomes the
// target page's title when the target is a page; raw HTML is escaped.
func (x *extraction) writeText(b *strings.Builder, l string, last, end int, code []bool, title, dest string) {
	ts := textStart(l, end, code)
	if title != "" && ts >= last && plain(l[ts+1:end]) == path.Base(strings.SplitN(strings.SplitN(dest, "#", 2)[0], "?", 2)[0]) {
		b.WriteString(escapeHTML(l[last:ts+1], 0))
		b.WriteString(mdtext.Text(title))
		return
	}
	b.WriteString(escapeHTML(l[last:end], 0))
}

// refDef resolves a reference definition's destination, l[start:end],
// and escapes raw HTML after it.
func (x *extraction) refDef(file, l string, start, end int) (string, error) {
	bare := strings.TrimSuffix(strings.TrimPrefix(l[start:end], "<"), ">")
	u, _, err := x.resolve(file, bare)
	if err != nil {
		return "", err
	}
	if u != bare {
		l = l[:start] + u + l[end:]
	}
	return escapeHTML(l, start+len(u)), nil
}

// escaped reports a byte preceded by an odd run of backslashes, which
// Markdown reads as the literal character.
func escaped(l string, i int) bool {
	n := 0
	for j := i - 1; j >= 0 && l[j] == '\\'; j-- {
		n++
	}
	return n%2 == 1
}

// destination finds a link destination starting at i, after "](": an
// angle-bracketed one or a run of non-blank characters with balanced
// parentheses. It returns the destination's bounds (the angle brackets
// left out), or -1 when there is none.
func destination(l string, i int) (start, end int) {
	if i < len(l) && l[i] == '<' {
		j := strings.IndexByte(l[i+1:], '>')
		if j < 0 {
			return -1, -1
		}
		return i + 1, i + 1 + j
	}
	depth := 0
	j := i
	for ; j < len(l); j++ {
		c := l[j]
		if c == ' ' || c == '\t' {
			break
		}
		if c == '(' {
			depth++
		}
		if c == ')' {
			if depth == 0 {
				break
			}
			depth--
		}
	}
	if j == i {
		return -1, -1
	}
	return i, j
}

// textStart finds the "[" that opens the link text closing at end (the
// "]"), outside code spans, or -1.
func textStart(l string, end int, code []bool) int {
	depth := 0
	for i := end - 1; i >= 0; i-- {
		if code[i] || escaped(l, i) {
			continue
		}
		switch l[i] {
		case ']':
			depth++
		case '[':
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

// plain is a link text as read: one code span around it removed.
func plain(s string) string {
	s = strings.TrimSpace(s)
	if n := runLen(s, 0); n > 0 && s[0] == '`' && len(s) > 2*n && strings.HasSuffix(s, s[:n]) {
		s = strings.TrimSpace(s[n : len(s)-n])
	}
	return s
}

// reScheme is a destination's URL scheme.
var reScheme = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9+.-]*):`)

// resolve resolves one link destination of file. A relative path becomes
// its section page's URL or its GitHub URL at the commit built, keeping a
// fragment; title is the page's title when the target is a section page
// named by its Markdown file. Anything else is returned as written: a
// fragment, a root-absolute path (the dialect lint checks its form) and an
// http, https or mailto URL. Another scheme, a path the commit does not
// have and a path climbing out of the repository are refused.
func (x *extraction) resolve(file, dest string) (u, title string, err error) {
	if dest == "" || strings.HasPrefix(dest, "#") || strings.HasPrefix(dest, "/") {
		return dest, "", nil
	}
	if m := reScheme.FindStringSubmatch(dest); m != nil {
		switch strings.ToLower(m[1]) {
		case "http", "https", "mailto":
			return dest, "", nil
		}
		return "", "", fmt.Errorf("link %q: the scheme %s: is not allowed; link with http, https or mailto", dest, m[1])
	}
	return x.relative(file, dest)
}

// relative resolves a relative link destination of file.
func (x *extraction) relative(file, dest string) (u, title string, err error) {
	p, frag, _ := strings.Cut(dest, "#")
	p, _, _ = strings.Cut(p, "?")
	if frag != "" {
		frag = "#" + frag
	}
	if p == "" {
		return frag, "", nil
	}
	up, err := url.PathUnescape(p)
	if err != nil {
		return "", "", fmt.Errorf("link %q: %w", dest, err)
	}
	t := path.Clean(path.Join(path.Dir(file), up))
	if t == ".." || strings.HasPrefix(t, "../") || path.IsAbs(t) {
		return "", "", fmt.Errorf("link %q climbs out of the repository (%s)", dest, t)
	}
	kind, ok := x.o.Paths[t]
	if !ok && t != "." {
		return "", "", fmt.Errorf("link %q names nothing in the repository at %s (%s)", dest, short(x.o.Commit), t)
	}
	if pg, ok := x.page(t); ok {
		if strings.HasSuffix(t, ".md") {
			title = pg.title
		}
		return pg.url + frag, title, nil
	}
	base := "https://github.com/" + x.o.Repo + "/"
	if t == "." {
		return base + "tree/" + x.o.Commit + frag, "", nil
	}
	return base + kind + "/" + x.o.Commit + "/" + escapePath(t) + frag, "", nil
}

// page is the section page a repository path names: an entry directory
// is its README's page, the entry root the section page.
func (x *extraction) page(t string) (target, bool) {
	if t == x.dir || t == "." && x.dir == "." {
		return x.pages[x.rel("INDEX.md")], true
	}
	if pg, ok := x.pages[t]; ok {
		return pg, true
	}
	pg, ok := x.pages[path.Join(t, "README.md")]
	return pg, ok
}

// escapePath escapes each segment of a repository path for a URL.
func escapePath(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return strings.Join(segs, "/")
}

func short(commit string) string {
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}

// reAutolink is a CommonMark autolink: a URI ("<https://...>", a scheme
// of 2 to 32 characters, no blank, control character, "<" or ">") or an
// email address in CommonMark's character set ("<name@host>").
var reAutolink = regexp.MustCompile("^<([A-Za-z][A-Za-z0-9+.-]{1,31}:[^<>\\x00-\\x20\\x7f]*|[A-Za-z0-9.!#$%&'*+/=?^_`{|}~-]+@[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?(\\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*)>")

// escapeHTML escapes, from byte from on, every "<" outside a code span
// that would open raw HTML (a tag, a closing tag, a comment or a
// declaration): the site renders raw HTML, and the text is shown as
// written instead. An autolink stays.
func escapeHTML(l string, from int) string {
	code := codeMask(l)
	var b strings.Builder
	for i := 0; i < len(l); i++ {
		c := l[i]
		if c == '<' && i >= from && !code[i] && !escaped(l, i) && i+1 < len(l) && opensHTML(l[i+1]) && !reAutolink.MatchString(l[i:]) {
			b.WriteString(`\<`)
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func opensHTML(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '/' || c == '!' || c == '?'
}
