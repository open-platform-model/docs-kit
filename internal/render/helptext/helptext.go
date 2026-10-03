// Package helptext turns a cobra command's help text (Short, Long, Example
// and flag usage, as Go source declares them) into Markdown in the page
// dialect: prose paragraphs, preformatted blocks, lists and examples, with
// code words in code spans and every other Markdown character escaped.
package helptext

import (
	"regexp"
	"strings"
	"unicode"
)

// Block is one part of a command's description: a prose paragraph, already
// formatted as Markdown, or a preformatted run of lines kept verbatim.
type Block struct {
	Pre   bool
	Lines []string
}

// Long is a command's Long help split into the description and the
// examples the help text lists under an "Examples:" line.
type Long struct {
	Desc     []Block
	Examples []string
}

// Parse splits a cobra Long text into prose paragraphs, preformatted
// blocks and examples.
//
// Long texts are raw Go strings, so every line after the first carries the
// indentation of the source that declares it (none, one tab, two tabs). The
// base indentation is the shallowest first line of any paragraph after the
// first; a line indented beyond the base is preformatted, a line at the base
// is prose. The first paragraph is always prose. A prose line reading
// "Examples:" starts the examples: the preformatted lines after it, up to the
// next prose line.
func Parse(long string) Long {
	lines := SplitLines(long)
	paras := paragraphs(lines)
	if len(paras) == 0 {
		return Long{}
	}
	base := baseIndent(lines, paras)

	var p Long
	p.Desc = append(p.Desc, Block{Lines: []string{joinProse(lines[paras[0][0]:paras[0][1]])}})

	b := &longBuilder{}
	for _, para := range paras[1:] {
		b.paragraphBreak()
		for _, line := range lines[para[0]:para[1]] {
			rel, rest := relIndent(line, base)
			if rel == "" {
				b.prose(rest)
				continue
			}
			b.preLine(rel + rest)
		}
	}
	b.flush()
	p.Desc = append(p.Desc, b.desc...)
	p.Examples = Dedent(b.examples)
	return p
}

// longBuilder accumulates the blocks of a Long text line by line.
type longBuilder struct {
	desc       []Block
	examples   []string
	inExamples bool
	cur        *Block // open prose or preformatted block
	gap        bool   // a blank line separates the next preformatted line
}

func (b *longBuilder) paragraphBreak() {
	if b.cur != nil && !b.cur.Pre {
		b.flush()
	}
	b.gap = true
}

func (b *longBuilder) prose(text string) {
	if b.cur != nil && b.cur.Pre {
		b.flush()
	}
	if t := strings.TrimSpace(text); t == "Examples:" || t == "Example:" {
		b.flush()
		b.inExamples = true
		b.gap = false
		return
	}
	b.inExamples = false
	if b.cur == nil {
		b.cur = &Block{}
	}
	b.cur.Lines = append(b.cur.Lines, text)
	b.gap = false
}

func (b *longBuilder) preLine(line string) {
	if b.inExamples {
		if b.gap && len(b.examples) > 0 {
			b.examples = append(b.examples, "")
		}
		b.examples = append(b.examples, line)
		b.gap = false
		return
	}
	if b.cur != nil && !b.cur.Pre {
		b.flush()
	}
	if b.cur == nil {
		b.cur = &Block{Pre: true}
	} else if b.gap {
		b.cur.Lines = append(b.cur.Lines, "")
	}
	b.cur.Lines = append(b.cur.Lines, line)
	b.gap = false
}

func (b *longBuilder) flush() {
	if b.cur == nil {
		return
	}
	if b.cur.Pre {
		b.desc = append(b.desc, Block{Pre: true, Lines: Dedent(b.cur.Lines)})
	} else {
		b.desc = append(b.desc, Block{Lines: []string{joinProse(b.cur.Lines)}})
	}
	b.cur = nil
}

// SplitLines normalizes line endings and drops trailing whitespace.
func SplitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	return lines
}

// paragraphs returns the [start, end) line ranges of the runs of non-blank lines.
func paragraphs(lines []string) [][2]int {
	var out [][2]int
	start := -1
	for i, l := range lines {
		switch {
		case l == "" && start >= 0:
			out = append(out, [2]int{start, i})
			start = -1
		case l != "" && start < 0:
			start = i
		}
	}
	if start >= 0 {
		out = append(out, [2]int{start, len(lines)})
	}
	return out
}

func indentOf(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// indentWidth measures indentation with a tab worth eight columns.
func indentWidth(indent string) int {
	w := 0
	for _, r := range indent {
		if r == '\t' {
			w += 8
		} else {
			w++
		}
	}
	return w
}

func baseIndent(lines []string, paras [][2]int) string {
	base, found := "", false
	for _, p := range paras[1:] {
		ind := indentOf(lines[p[0]])
		if !found || indentWidth(ind) < indentWidth(base) {
			base, found = ind, true
		}
	}
	return base
}

// relIndent returns the indentation of line beyond base and the rest of the
// line. A line not indented by base counts as prose.
func relIndent(line, base string) (rel, rest string) {
	ind := indentOf(line)
	rest = line[len(ind):]
	if !strings.HasPrefix(ind, base) {
		return "", rest
	}
	return ind[len(base):], rest
}

// Dedent removes the indentation every non-blank line shares.
func Dedent(lines []string) []string {
	common, found := "", false
	for _, l := range lines {
		if l == "" {
			continue
		}
		ind := indentOf(l)
		if !found {
			common, found = ind, true
			continue
		}
		for !strings.HasPrefix(ind, common) {
			common = common[:len(common)-1]
		}
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = strings.TrimPrefix(l, common)
	}
	return out
}

func joinProse(lines []string) string {
	parts := make([]string, 0, len(lines))
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}

// Prose turns one line of plain help text into Markdown. Words that
// name code (flags, paths, placeholders, environment variables, CUE
// definitions, file names) and single-quoted spans become code spans, so the
// site never reads them as markup or links; neighboring code words separated
// by one space share one span (--platform <dir>). Every other Markdown
// character is escaped. The words themselves are unchanged.
func Prose(s string) string {
	var pieces []piece
	rest := strings.TrimSpace(s)
	for rest != "" {
		if p, after, ok := quotedSpan(rest); ok {
			pieces = appendPiece(pieces, p)
			rest = strings.TrimLeft(after, " \t")
			continue
		}
		word := rest
		if i := strings.IndexAny(rest, " \t"); i >= 0 {
			word, rest = rest[:i], strings.TrimLeft(rest[i:], " \t")
		} else {
			rest = ""
		}
		pieces = appendPiece(pieces, formatWord(word))
	}
	out := make([]string, len(pieces))
	for i, p := range pieces {
		out[i] = p.String()
	}
	if len(out) > 0 {
		out[0] = escapeLineStart(out[0])
	}
	return strings.Join(out, " ")
}

// piece is one formatted word: plain text, or a code span with the
// punctuation around it.
type piece struct {
	text              string // plain text, unescaped
	lead, code, trail string // a code span and its surrounding punctuation
	isCode            bool
}

func (p piece) String() string {
	if !p.isCode {
		return escapeText(p.text)
	}
	return escapeText(p.lead) + CodeSpan(p.code) + escapeText(p.trail)
}

// appendPiece appends p, joining it to the previous code span when nothing
// but one space separates them.
func appendPiece(pieces []piece, p piece) []piece {
	if n := len(pieces); n > 0 && p.isCode && p.lead == "" && pieces[n-1].isCode && pieces[n-1].trail == "" {
		pieces[n-1].code += " " + p.code
		pieces[n-1].trail = p.trail
		return pieces
	}
	return append(pieces, p)
}

// quotedSpan recognizes a single-quoted span at the start of s, such as
// 'opm module vet', as a code span with any trailing punctuation after it.
func quotedSpan(s string) (p piece, after string, ok bool) {
	if !strings.HasPrefix(s, "'") || len(s) < 3 || s[1] == ' ' {
		return piece{}, "", false
	}
	end := strings.Index(s[1:], "'")
	if end <= 0 {
		return piece{}, "", false
	}
	end++
	inner := s[1:end]
	if strings.HasSuffix(inner, " ") {
		return piece{}, "", false
	}
	tail := s[end+1:]
	punct := tail[:len(tail)-len(strings.TrimLeft(tail, ".,;:!?)"))]
	next := tail[len(punct):]
	if next != "" && next[0] != ' ' && next[0] != '\t' {
		return piece{}, "", false
	}
	return piece{code: inner, trail: punct, isCode: true}, next, true
}

var (
	reFlag    = regexp.MustCompile(`^--?[A-Za-z]`)
	rePlus    = regexp.MustCompile(`^\+[a-z]`)
	reDef     = regexp.MustCompile(`^#[A-Za-z]`)
	reFile    = regexp.MustCompile(`^[A-Za-z0-9_.-]+\.(cue|yaml|yml|json|md|go|sh|toml)$`)
	reOrdinal = regexp.MustCompile(`^\d+[.)]`)
)

// formatWord classifies one whitespace-separated word, keeping surrounding
// punctuation outside a code span.
func formatWord(w string) piece {
	core := strings.TrimLeft(w, `("`)
	lead := w[:len(w)-len(core)]
	trimmed := strings.TrimRight(core, `.,;:!?)"`)
	trail := core[len(trimmed):]
	if trimmed == "" || !isCode(trimmed) {
		return piece{text: w}
	}
	if strings.HasPrefix(lead, `"`) && strings.HasPrefix(trail, `"`) {
		// A quoted code word: the code span replaces the quotes.
		lead, trail = lead[:len(lead)-1], trail[1:]
	}
	return piece{lead: lead, code: trimmed, trail: trail, isCode: true}
}

func isCode(w string) bool {
	switch {
	case reFlag.MatchString(w), rePlus.MatchString(w), reDef.MatchString(w), reFile.MatchString(w):
		return true
	case strings.ContainsAny(w, "@_=~<>[]{}*`|\\$"):
		return strings.IndexFunc(w, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) >= 0
	case strings.Contains(w, "/"):
		return isPath(w)
	}
	return false
}

// isPath tells a path ("cue.mod/module.cue", "./src", "platform/") from a
// word pair joined by a slash ("beta/GA").
func isPath(w string) bool {
	return strings.Contains(w, ".") || strings.HasPrefix(w, "/") || strings.HasSuffix(w, "/")
}

var textEscaper = strings.NewReplacer(
	`\`, `\\`, "`", "\\`", `*`, `\*`, `_`, `\_`, `[`, `\[`, `]`, `\]`,
	`<`, `&lt;`, `>`, `&gt;`, `&`, `&amp;`, `#`, `\#`, `~`, `\~`, `|`, `\|`,
)

func escapeText(s string) string { return textEscaper.Replace(s) }

// escapeLineStart keeps a paragraph's first word from reading as a list item.
func escapeLineStart(w string) string {
	switch {
	case w == "-" || w == "+":
		return `\` + w
	case reOrdinal.MatchString(w):
		i := strings.IndexAny(w, ".)")
		return w[:i] + `\` + w[i:]
	}
	return w
}

func CodeSpan(s string) string {
	ticks := "`"
	for strings.Contains(s, ticks) {
		ticks += "`"
	}
	if ticks == "`" {
		return "`" + s + "`"
	}
	return ticks + " " + s + " " + ticks
}

// Fence renders lines as a fenced code block tagged lang, with a fence longer
// than any backtick run at the start of a line.
func Fence(lang string, lines []string) string {
	ticks := "```"
	for _, l := range lines {
		for strings.HasPrefix(strings.TrimLeft(l, " "), ticks) {
			ticks += "`"
		}
	}
	return ticks + lang + "\n" + strings.Join(lines, "\n") + "\n" + ticks + "\n"
}

// FenceLang tags a preformatted block sh when it holds commands of the CLI
// named cli and nothing but commands and comments, and text otherwise.
func FenceLang(lines []string, cli string) string {
	sawCommand := false
	for _, l := range lines {
		switch {
		case l == "" || strings.HasPrefix(l, "# "):
		case strings.HasPrefix(l, cli+" "):
			sawCommand = true
		default:
			return "text"
		}
	}
	if sawCommand {
		return "sh"
	}
	return "text"
}

// EscapeShortcodes keeps Hugo from expanding shortcode delimiters, which it
// does even inside code fences and code spans.
func EscapeShortcodes(s string) string {
	return strings.NewReplacer("{{<", "{{</*", ">}}", "*/>}}", "{{%", "{{%/*", "%}}", "*/%}}").Replace(s)
}

// WriteBlocks writes a parsed description: prose paragraphs through
// Prose and then clean, a preformatted block whose every line is one list
// item as a Markdown list, any other preformatted block as a fence tagged
// by FenceLang for the CLI named cli. Blocks are separated by one blank line.
func WriteBlocks(b *strings.Builder, blocks []Block, cli string, clean func(string) string) {
	for i, bl := range blocks {
		if i > 0 {
			b.WriteString("\n")
		}
		if marker, items, ok := ListItems(bl.Lines); bl.Pre && ok {
			for _, item := range items {
				b.WriteString(marker + " " + clean(Prose(item)) + "\n")
			}
			continue
		}
		if bl.Pre {
			b.WriteString(Fence(FenceLang(bl.Lines, cli), bl.Lines))
			continue
		}
		b.WriteString(clean(Prose(bl.Lines[0])) + "\n")
	}
}

var reListItem = regexp.MustCompile(`^(-|\d+\.) +(\S.*)$`)

// ListItems reads a preformatted block in which every line is one bullet
// ("- item") or one numbered item ("1. item") as a Markdown list: marker is
// "-" or "1.", the marker every item of an ordered list carries. A block with
// a continuation line, a blank line or mixed markers is not a list.
func ListItems(lines []string) (marker string, items []string, ok bool) {
	for _, l := range lines {
		m := reListItem.FindStringSubmatch(l)
		if m == nil {
			return "", nil, false
		}
		mk := "-"
		if m[1] != "-" {
			mk = "1."
		}
		if marker != "" && mk != marker {
			return "", nil, false
		}
		marker = mk
		items = append(items, m[2])
	}
	return marker, items, len(items) > 0
}

// Cell keeps a value inside its table column and on its row: pipes are
// escaped and line breaks become spaces.
func Cell(s string) string {
	s = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(s)
	return strings.ReplaceAll(strings.ReplaceAll(s, "|", `\|`), `\\|`, `\|`)
}

// Sentence ends a summary with a full stop unless it already ends a sentence.
func Sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasSuffix(s, ".") || strings.HasSuffix(s, "?") || strings.HasSuffix(s, "!") {
		return s
	}
	return s + "."
}

// Anchor is the heading id Hugo gives a heading such as "opm module apply".
func Anchor(heading string) string {
	return strings.ReplaceAll(strings.ToLower(heading), " ", "-")
}

// YAMLQuote quotes a front-matter value as a YAML double-quoted scalar,
// its whitespace collapsed.
func YAMLQuote(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
