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

var crdTemplates = template.Must(template.New("").Funcs(template.FuncMap{
	"inline":    func(s string) string { return crdInline(s, false) },
	"cell":      func(s string) string { return crdInline(s, true) },
	"code":      crdCode,
	"codeList":  crdCodeList,
	"ruleField": crdRuleField,
	"ruleText":  crdRuleText,
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
	return []Page{{Path: m.Page.Path, Body: body, Completable: true, Heading: "## " + m.Kinds[0].Kind, Tail: tail}}, nil
}

var reBlankRuns = regexp.MustCompile(`\n{3,}`)

// CRDEntries renders one entry per kind, in model order: the generated
// body of the crd page, without front matter. A part with nothing to show
// is left out; parts are separated by one blank line and the body ends in
// one newline.
func CRDEntries(m *crd.Model) (string, error) {
	var b bytes.Buffer
	if err := crdTemplates.ExecuteTemplate(&b, "entries.md.tmpl", m); err != nil {
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
	"~", `\~`, "|", `\|`, "#", `\#`, "{{", `{\{`,
)

// reDecisionLink is the link the "link" citation policy writes into the
// doc model; the renderer keeps it as written.
var reDecisionLink = regexp.MustCompile(`\[\d{4}:D[0-9DR:/]*\]\(/enhancements/\d{4}/decisions/\)`)

// crdProse escapes prose outside code spans, keeping decision links.
func crdProse(s string) string {
	var b strings.Builder
	last := 0
	for _, m := range reDecisionLink.FindAllStringIndex(s, -1) {
		b.WriteString(crdEscaper.Replace(s[last:m[0]]))
		b.WriteString(s[m[0]:m[1]])
		last = m[1]
	}
	b.WriteString(crdEscaper.Replace(s[last:]))
	return b.String()
}

// crdInline renders doc-model prose as Markdown inline content: backtick
// spans stay code, everything else is escaped. An unpaired backtick makes
// the whole text prose, the backtick escaped too. In a table cell a pipe
// is escaped inside code spans as well.
func crdInline(s string, inTable bool) string {
	parts := strings.Split(s, "`")
	if len(parts)%2 == 0 {
		return strings.ReplaceAll(crdProse(s), "`", "\\`")
	}
	var b strings.Builder
	for i, part := range parts {
		if i%2 == 1 {
			b.WriteString(crdCodeIn(part, inTable))
			continue
		}
		b.WriteString(crdProse(part))
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
func crdRuleText(r crd.Rule) string {
	s := r.Rule
	if len(r.Values) > 0 {
		s += " " + crdCodeList(r.Values)
	}
	if r.Message != "" {
		s += "; refused with: " + crdInline(r.Message, true)
	}
	return s
}
