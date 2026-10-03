package render

import (
	"bytes"
	"embed"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"github.com/open-platform-model/docs-kit/internal/extract/crd"
	"github.com/open-platform-model/docs-kit/internal/mdtext"
)

//go:embed templates/crd/*.tmpl
var crdTemplateFiles embed.FS

// crdFuncs are the crd template's helpers; inline and cell are bound to
// the model's citation policy per render.
func crdFuncs(link bool) template.FuncMap {
	return template.FuncMap{
		"inline":   func(s string) string { return crdInline(s, false, link) },
		"cell":     func(s string) string { return crdInline(s, true, link) },
		"fence":    crdFence,
		"ruleText": func(r crd.Rule) string { return crdRuleText(r, link) },
	}
}

var crdTemplates = template.Must(template.New("").Funcs(crdFuncs(false)).Funcs(template.FuncMap{
	"code":      crdCode,
	"codeList":  crdCodeList,
	"ruleField": crdRuleField,
	"fields": func(title string, rows []crd.Field) any {
		return struct {
			Title   string
			Rows    []crd.Field
			Default bool
		}{title, rows, slices.ContainsFunc(rows, func(f crd.Field) bool { return f.Default != nil })}
	},
}).ParseFS(crdTemplateFiles, "templates/crd/*.tmpl"))

// crdRenderer renders a crd data file: one completable page holding an
// entry per kind.
type crdRenderer struct{}

func (crdRenderer) Schema() string { return crd.SchemaID }

func (crdRenderer) Render(data []byte, _ Target) ([]Page, error) {
	m, err := crd.Decode(data)
	if err != nil {
		return nil, err
	}
	if len(m.Kinds) == 0 {
		return nil, fmt.Errorf("data/%s lists no kind", crd.DataFile)
	}
	tail, err := CRDEntries(m)
	if err != nil {
		return nil, err
	}
	var fm strings.Builder
	fm.WriteString("---\n")
	fmt.Fprintf(&fm, "title: %s\n", mdtext.YAMLString(m.Page.Title))
	fmt.Fprintf(&fm, "description: %s\n", mdtext.YAMLString(m.Page.Description))
	fm.WriteString("type: reference\n")
	if m.Page.Weight != nil {
		fmt.Fprintf(&fm, "weight: %d\n", *m.Page.Weight)
	}
	fm.WriteString("---\n\n")
	body := fm.String() + tail
	if err := mdtext.CheckShortcodes(m.Page.Path, body); err != nil {
		return nil, err
	}
	// An authored page that already holds any kind's heading is refused,
	// not only the first's.
	var more []string
	for i := 1; i < len(m.Kinds); i++ {
		more = append(more, "## "+m.Kinds[i].Kind)
	}
	return []Page{{Path: m.Page.Path, Body: body, Completable: true, Heading: "## " + m.Kinds[0].Kind, Headings: more, Tail: tail}}, nil
}

var reBlankRuns = regexp.MustCompile(`\n{3,}`)

// CRDEntries renders one entry per kind, in model order: the generated
// body of the crd page, without front matter. A part with nothing to show
// is left out; parts are separated by one blank line and the body ends in
// one newline.
func CRDEntries(m *crd.Model) (string, error) {
	t, err := crdTemplates.Clone()
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := t.Funcs(crdFuncs(m.Citations == crd.CitationsLink)).ExecuteTemplate(&b, "entries.md.tmpl", m); err != nil {
		return "", err
	}
	s := reBlankRuns.ReplaceAllString(b.String(), "\n\n")
	return strings.Trim(s, "\n") + "\n", nil
}

// crdEscaper escapes what Markdown or Hugo would read as markup in prose
// taken from a CRD: the common set of every renderer, plus "~" and "#",
// which a schema description uses ("#Platform") and which would otherwise
// start a strike-through or a heading.
var crdEscaper = strings.NewReplacer(
	`\`, `\\`, "*", `\*`, "_", `\_`, "[", `\[`, "]", `\]`, "<", `\<`, ">", `\>`,
	"~", `\~`, "|", `\|`, "#", `\#`,
)

// reCitation is an enhancement decision citation: "0015:D3",
// "0015:D3/D16", "0011:D9:R2". Escaping leaves it intact.
var reCitation = regexp.MustCompile(`\b(\d{4}):D\d+(?::R\d+(?:/R\d+)*)?(?:/D\d+(?::R\d+(?:/R\d+)*)?)*`)

// crdProse escapes prose outside code spans and, under the link policy,
// links each decision citation to its enhancement's decisions page. The
// model's text is plain, so the only links on the page are the ones built
// here from a citation.
func crdProse(s string, link bool) string {
	s = crdEscaper.Replace(s)
	// "{{{" escapes to "{\{{", which still opens one: repeat until none is
	// left.
	for strings.Contains(s, "{{") {
		s = strings.ReplaceAll(s, "{{", `{\{`)
	}
	if link {
		s = reCitation.ReplaceAllString(s, "[$0](/enhancements/$1/decisions/)")
	}
	return s
}

// crdInline renders doc-model prose as Markdown inline content: backtick
// spans stay code, everything else is escaped. An unpaired backtick makes
// the whole text prose, the backtick escaped too. In a table cell a pipe
// is escaped inside code spans as well.
func crdInline(s string, inTable, link bool) string {
	parts := strings.Split(s, "`")
	if len(parts)%2 == 0 {
		return strings.ReplaceAll(crdProse(s, link), "`", "\\`")
	}
	var b strings.Builder
	for i, part := range parts {
		if i%2 == 1 {
			b.WriteString(crdCodeIn(part, inTable))
			continue
		}
		b.WriteString(crdProse(part, link))
	}
	return b.String()
}

// crdCode is a code span in a table cell.
func crdCode(s string) string { return crdCodeIn(s, true) }

func crdCodeIn(s string, inTable bool) string {
	if inTable {
		s = mdtext.Cell(s)
	}
	return mdtext.Code(s)
}

func crdCodeList(items []string) string {
	out := make([]string, 0, len(items))
	for _, i := range items {
		out = append(out, crdCode(i))
	}
	return strings.Join(out, ", ")
}

func crdRuleField(path string) string {
	if path == "" {
		return "the object"
	}
	return crdCode(path)
}

// crdRuleText is a rule's sentence: "One of `a`, `b`", "CEL rule `x`;
// refused with: <message>".
func crdRuleText(r crd.Rule, link bool) string {
	s := r.Rule
	if len(r.Values) > 0 {
		s += " " + crdCodeList(r.Values)
	}
	if r.Message != "" {
		s += "; refused with: " + crdInline(r.Message, true, link)
	}
	return s
}

var reFenceRun = regexp.MustCompile("`+|~+")

// crdFence is a backtick fence longer than any backtick or tilde run in
// text, at least three, so no line of a sample can close it.
func crdFence(text string) string {
	n := 3
	for _, r := range reFenceRun.FindAllString(text, -1) {
		if len(r) >= n {
			n = len(r) + 1
		}
	}
	return strings.Repeat("`", n)
}
