package render

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/open-platform-model/docs-kit/internal/extract/cuedefs"
	"github.com/open-platform-model/docs-kit/internal/mdtext"
)

// pagesHeading opens the definitions index's generated tail, which an
// authored index may complete.
const pagesHeading = "## Pages"

// defsRenderer renders a cue-definitions data file: the section index and
// one page per configured group (docs-kit C17).
type defsRenderer struct{}

func (defsRenderer) Schema() string { return cuedefs.SchemaID }

func (defsRenderer) Render(data []byte, t Target) ([]Page, error) {
	m, err := cuedefs.Decode(data)
	if err != nil {
		return nil, err
	}
	return Definitions(m, t)
}

// Definitions renders the section index and every page of a definitions
// model. Its pages link one another under /docs/, so it renders only into
// a docs bundle.
func Definitions(m *cuedefs.Model, t Target) ([]Page, error) {
	if t.Kind != KindDocs {
		return nil, fmt.Errorf("cue-definitions renders reference pages under /docs/; give the bundle placement kind \"docs\"")
	}
	r := &defsRender{m: m, t: t, byName: map[string]*cuedefs.Definition{}, pageTitle: map[string]string{}}
	for i := range m.Definitions {
		r.byName[m.Definitions[i].Name] = &m.Definitions[i]
	}
	for _, p := range m.Pages {
		r.pageTitle[p.File] = p.Title
	}
	if err := checkLinks(m); err != nil {
		return nil, err
	}
	index, err := r.index()
	if err != nil {
		return nil, err
	}
	pages := []Page{index}
	for _, p := range m.Pages {
		body, err := r.page(p)
		if err != nil {
			return nil, fmt.Errorf("%s%s: %w", m.Section, p.File, err)
		}
		pages = append(pages, Page{Path: m.Section + p.File + ".md", Body: body})
	}
	for _, p := range pages {
		if err := mdtext.CheckShortcodes(p.Path, p.Body); err != nil {
			return nil, err
		}
	}
	return pages, nil
}

type defsRender struct {
	m         *cuedefs.Model
	t         Target
	byName    map[string]*cuedefs.Definition
	pageTitle map[string]string
}

// indexURL is the section index's URL, "/docs/reference/definitions/".
func (r *defsRender) indexURL() string {
	return r.t.URL(strings.TrimSuffix(r.m.Section, "/"))
}

// link is the URL of a placed definition's entry, or "" when the pages
// place none of that name.
func (r *defsRender) link(name string) string {
	d, ok := r.byName[name]
	if !ok {
		return ""
	}
	return r.t.URL(r.m.Section+d.Page) + "#" + d.Anchor
}

func (r *defsRender) linkList(names []string) string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = "[`" + n + "`](" + r.link(n) + ")"
	}
	return strings.Join(out, ", ")
}

func (r *defsRender) index() (Page, error) {
	var b strings.Builder
	b.WriteString(pagesHeading + "\n\n")
	for _, p := range r.m.Pages {
		fmt.Fprintf(&b, "- [%s](%s): %s\n", p.Title, r.t.URL(r.m.Section+p.File), p.Description)
	}
	b.WriteString("\n## All definitions\n\n")
	b.WriteString("| Definition | Page | Summary |\n| --- | --- | --- |\n")
	for _, p := range r.m.Pages {
		for _, n := range p.Definitions {
			d := r.byName[n]
			sum := strings.ReplaceAll(r.markdown(d.Summary, n), "|", `\|`)
			fmt.Fprintf(&b, "| [`%s`](%s) | %s | %s |\n", n, r.link(n), p.Title, sum)
		}
	}
	tail := b.String()
	intro := r.m.Intro
	if intro == "" {
		intro = "Every definition below belongs to " + mdtext.Code(r.m.ModulePath) + "."
	}
	body, err := execute("defs-index.md.tmpl", map[string]any{
		"Title": r.m.Title, "Description": r.m.Description, "Weight": r.m.Weight,
		"Intro": intro, "Tail": strings.TrimRight(tail, "\n"),
	})
	if err != nil {
		return Page{}, err
	}
	return Page{Path: r.m.Section + "_index.md", Body: body, Completable: true, Heading: pagesHeading, Tail: tail}, nil
}

func (r *defsRender) page(p cuedefs.Page) (string, error) {
	entries := make([]string, 0, len(p.Definitions))
	for _, n := range p.Definitions {
		d, ok := r.byName[n]
		if !ok {
			return "", fmt.Errorf("page lists %s, which the model does not define", n)
		}
		entries = append(entries, r.entry(d))
	}
	return execute("defs-page.md.tmpl", map[string]any{
		"Title": p.Title, "Description": p.Description, "Weight": p.Weight,
		"OnPage": r.linkList(p.Definitions), "Index": r.indexURL(), "Entries": entries,
	})
}

// entry writes one definition in a fixed order: summary, at a glance,
// spec, example, notes, enforcement.
func (r *defsRender) entry(d *cuedefs.Definition) string {
	md := func(s string) string { return r.markdown(s, d.Name) }
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\n", d.Name)
	if d.Summary != "" {
		b.WriteString(md(d.Summary) + "\n\n")
	}
	b.WriteString("**At a glance**\n\n")
	fmt.Fprintf(&b, "- Source: %s in %s\n", mdtext.Code(d.File), mdtext.Code(r.m.ModulePath))
	fmt.Fprintf(&b, "- Shape: %s\n", d.Shape)
	if len(d.Embeds) > 0 {
		fmt.Fprintf(&b, "- Embeds: %s\n", r.linkList(d.Embeds))
	}
	if len(d.Uses) > 0 {
		fmt.Fprintf(&b, "- Uses: %s\n", r.linkList(d.Uses))
	}
	if len(d.UsedBy) > 0 {
		fmt.Fprintf(&b, "- Used by: %s\n", r.linkList(d.UsedBy))
	}
	f := fence(d.CUE)
	fmt.Fprintf(&b, "\n**Spec**\n\n%scue\n%s\n%s\n", f, d.CUE, f)
	if len(d.Example) > 0 {
		f := fence(strings.Join(d.Example, "\n"))
		b.WriteString("\n**Example**\n\n" + f + "cue\n")
		for _, l := range d.Example {
			b.WriteString(l + "\n")
		}
		b.WriteString(f + "\n")
	}
	if len(d.Notes) > 0 {
		b.WriteString("\n**Notes**\n\n")
		b.WriteString(notesMarkdown(d.Notes, md))
	}
	if len(d.Rules) > 0 {
		fmt.Fprintf(&b, "\n**Enforcement**\n\nCUE enforces each of these rules on a value unified with `%s`:\n\n", d.Name)
		for _, rl := range d.Rules {
			fmt.Fprintf(&b, "- %s\n", md(rl.Rule))
			if rl.Code != "" {
				f := fence(rl.Code)
				fmt.Fprintf(&b, "\n  %stext\n  %s\n  %s\n\n", f, rl.Code, f)
			}
		}
	}
	return b.String()
}

// notesMarkdown writes note blocks as Markdown: a paragraph as prose, a
// "- " entry as a list item, a run of indented entries as a text block,
// and "" as a break that ends a text block.
func notesMarkdown(notes []string, md func(string) string) string {
	var b strings.Builder
	prevList := false
	for i := 0; i < len(notes); i++ {
		n := notes[i]
		switch {
		case n == "":
			continue
		case strings.HasPrefix(n, "- "):
			if !prevList && b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString("- " + md(strings.TrimPrefix(n, "- ")) + "\n")
			prevList = true
			continue
		case strings.HasPrefix(n, "  "):
			var lines []string
			for ; i < len(notes) && strings.HasPrefix(notes[i], "  "); i++ {
				lines = append(lines, strings.TrimSpace(notes[i]))
			}
			i--
			text := strings.Join(lines, "\n")
			f := fence(text)
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(f + "text\n" + text + "\n" + f + "\n")
		default:
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(md(n) + "\n")
		}
		prevList = false
	}
	return b.String()
}

// reDollarRef matches a "$"-prefixed field name in prose: "$secretName".
var reDollarRef = regexp.MustCompile(`\$[A-Za-z_][A-Za-z0-9_]*`)

// reDefRef matches a definition reference in prose: "#Trait",
// "#Trait.optional", "#ctx.components".
var reDefRef = regexp.MustCompile(`#[A-Za-z_][A-Za-z0-9_]*(?:\.[#$A-Za-z_][A-Za-z0-9_]*)*`)

// markdown renders one line of definition prose. A Markdown link keeps its
// URL as written and has its text escaped; outside code spans and links it
// links a reference to a placed definition other than
// self, puts any other "#name" and "$name" in a code span, and escapes "<"
// and "{{" so no text reads as raw HTML or a shortcode. The prose is
// Markdown as written in the doc comment, so nothing else is escaped.
func (r *defsRender) markdown(s, self string) string {
	var b strings.Builder
	for _, seg := range proseSegments(s) {
		switch {
		case seg.verbatim:
			b.WriteString(seg.text)
		case seg.link:
			b.WriteString(reMDLink.ReplaceAllStringFunc(seg.text, func(l string) string {
				g := reMDLink.FindStringSubmatch(l)
				return "[" + escapeProse(g[1]) + "](" + g[2] + g[3] + ")"
			}))
		default:
			b.WriteString(r.prose(seg.text, self))
		}
	}
	return b.String()
}

// escapeProse escapes "<" and "{{", so no text reads as raw HTML or a
// shortcode.
func escapeProse(p string) string {
	p = strings.ReplaceAll(p, "<", `\<`)
	return strings.ReplaceAll(p, "{{", `{\{`)
}

func (r *defsRender) prose(p, self string) string {
	p = escapeProse(p)
	p = reDefRef.ReplaceAllStringFunc(p, func(m string) string {
		base := m
		if j := strings.Index(m, "."); j > 0 {
			base = m[:j]
		}
		if base != self {
			if u := r.link(base); u != "" {
				return "[`" + m + "`](" + u + ")"
			}
		}
		return "`" + m + "`"
	})
	return reDollarRef.ReplaceAllString(p, "`$0`")
}

type segment struct {
	text     string
	verbatim bool // a code span, written as it is
	link     bool // a Markdown link: its text escaped, its URL as written
}

// reMDLink matches an inline Markdown link, "[text](url)" or
// "[text](url "title")": text, URL and title are its groups.
var reMDLink = regexp.MustCompile(`\[([^\]\n]*)\]\(([^)\s]*)(\s+"[^"\n]*")?\)`)

// reScheme matches a URL that names a scheme.
var reScheme = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9+.-]*):`)

// checkLinks refuses a Markdown link in a definition's prose whose URL
// names a scheme other than http or https: the site renders raw HTML and
// links as written, so a javascript: link would run.
func checkLinks(m *cuedefs.Model) error {
	for i := range m.Definitions {
		d := &m.Definitions[i]
		texts := append([]string{d.Summary}, d.Notes...)
		for _, rl := range d.Rules {
			texts = append(texts, rl.Rule)
		}
		for _, t := range texts {
			for _, seg := range proseSegments(t) {
				if !seg.link {
					continue
				}
				u := reMDLink.FindStringSubmatch(seg.text)[2]
				if sc := reScheme.FindStringSubmatch(u); len(sc) == 2 && !strings.EqualFold(sc[1], "http") && !strings.EqualFold(sc[1], "https") {
					return fmt.Errorf("%s (%s): the doc comment links %q; a link is http, https or relative", d.Name, d.File, u)
				}
			}
		}
	}
	return nil
}

// proseSegments splits prose into code spans (a run of n backticks up to
// the next run of exactly n), Markdown links and the text between them.
func proseSegments(s string) []segment {
	var out []segment
	text := func(t string) {
		for t != "" {
			loc := reMDLink.FindStringIndex(t)
			if loc == nil {
				out = append(out, segment{text: t})
				return
			}
			if loc[0] > 0 {
				out = append(out, segment{text: t[:loc[0]]})
			}
			out = append(out, segment{text: t[loc[0]:loc[1]], link: true})
			t = t[loc[1]:]
		}
	}
	start := 0
	for i := 0; i < len(s); {
		if s[i] != '`' {
			i++
			continue
		}
		n := backtickRun(s, i)
		end := closingRun(s, i+n, n)
		if end < 0 {
			i += n // an unmatched run is text
			continue
		}
		text(s[start:i])
		out = append(out, segment{text: s[i : end+n], verbatim: true})
		i = end + n
		start = i
	}
	text(s[start:])
	return out
}

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

// fence is a code fence longer than any backtick run in code, at least
// three backticks.
func fence(code string) string {
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
