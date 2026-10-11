package enhancements

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/open-platform-model/docs-kit/internal/mdtext"
)

// capability is one main spec, <specs>/<name>/spec.md.
type capability struct {
	name, file, text string
	title, summary   string
}

// capabilityURL is the page of a capability.
func capabilityURL(name string) string { return "/enhancements/capabilities/" + name + "/" }

// readCapabilities reads the main specs, in name order, and registers
// their files and the specs directory as link targets. No specs directory
// holds none.
func (x *extraction) readCapabilities() error {
	dir := path.Join(x.dir, dirOr(x.o.Specs, defaultSpecs))
	ents, err := os.ReadDir(filepath.Join(x.o.Root, filepath.FromSlash(dir)))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := checkDir(x.o.Root, dir, dir); err != nil {
		return err
	}
	for _, e := range ents {
		sub := path.Join(dir, e.Name())
		if e.Type()&fs.ModeSymlink != 0 {
			return checkDir(x.o.Root, sub, sub)
		}
		if !e.IsDir() {
			continue
		}
		if !reCap.MatchString(e.Name()) {
			return fmt.Errorf("%s is not a capability name; a capability is a lower-case word, with hyphens", sub)
		}
		file := path.Join(sub, "spec.md")
		text, err := readFile(x.o.Root, file)
		if err != nil {
			return err
		}
		c := capability{name: e.Name(), file: file, text: text, title: capabilityTitle(e.Name()), summary: purposeSentence(text, e.Name())}
		x.caps = append(x.caps, c)
		x.pages[file] = target{url: capabilityURL(c.name), title: c.title}
	}
	if len(x.caps) > 0 {
		x.pages[dir] = target{url: "/enhancements/capabilities/", title: "the capabilities"}
	}
	return nil
}

// capabilityTitle is a capability's name as words: "single-build-render"
// is "Single build render".
func capabilityTitle(name string) string {
	s := strings.ReplaceAll(name, "-", " ")
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

// purposeSentence is the first sentence of a spec's "## Purpose" section,
// whitespace collapsed; a spec with no Purpose gets a plain sentence.
func purposeSentence(text, name string) string {
	var para []string
	in := false
	for _, l := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(l, "## ") {
			if in {
				break
			}
			in = strings.TrimSpace(strings.TrimPrefix(l, "## ")) == "Purpose"
			continue
		}
		if !in {
			continue
		}
		if strings.TrimSpace(l) == "" {
			if len(para) > 0 {
				break
			}
			continue
		}
		para = append(para, l)
	}
	s := collapse(strings.Join(para, " "))
	if s == "" {
		return "The capability " + name + "."
	}
	if i := strings.Index(s, ". "); i >= 0 {
		return s[:i+1]
	}
	return s
}

// changeRow is one archived change that touched a capability.
type changeRow struct {
	date, id string // id is "" for a seed
}

// capabilityPages adds capabilities/_index.md and one page for each main
// spec. Each page ends with the archived changes that touched the
// capability: a seed, or an entry that was delivered (the deltas of a
// rejected, withdrawn or superseded entry never reached the main spec).
func (x *extraction) capabilityPages(m *Model) error {
	if len(x.caps) == 0 {
		return nil
	}
	history, err := x.capabilityHistory(m)
	if err != nil {
		return err
	}
	title := map[string]string{}
	for i := range m.Entries {
		title[m.Entries[i].ID] = m.Entries[i].Title
	}
	var rows strings.Builder
	rows.WriteString("| Capability | Summary |\n| --- | --- |\n")
	var inputs []string
	for i := range x.caps {
		c := &x.caps[i]
		body := c.text
		if hist := history[c.name]; len(hist) > 0 {
			body = strings.TrimRight(body, "\n") + "\n\n## Entries that changed it\n\n| Entry | Date |\n| --- | --- |\n"
			for _, r := range hist {
				entry := "seed"
				if r.id != "" {
					entry = "[" + mdtext.Cell(mdtext.Text(r.id+": "+title[r.id])) + "](/enhancements/" + r.id + "/)"
				}
				body += "| " + entry + " | " + r.date + " |\n"
			}
		}
		x.out = append(x.out, Page{Path: "capabilities/" + c.name + ".md", Title: c.title, Description: c.summary, Type: "explanation", Body: body, Source: c.file})
		rows.WriteString("| [" + mdtext.Cell(mdtext.Text(c.title)) + "](" + capabilityURL(c.name) + ") | " + mdtext.Cell(mdtext.Text(c.summary)) + " |\n")
		inputs = append(inputs, c.file)
	}
	x.pages[path.Join(x.dir, "capabilities")] = target{url: "/enhancements/capabilities/", title: "the capabilities"}
	x.out = append(x.out, Page{
		Path: "capabilities/_index.md", Title: "Capabilities", Inputs: inputs,
		Description: "What OPM specifies, one page for each capability, with the entries that changed it.",
		Body:        "Each capability is a set of requirements that hold today. A page lists the entries that changed it.\n\n" + rows.String(),
	})
	return nil
}

// capabilityHistory lists, for each capability, the archived changes whose
// delta specs touch it, by date and then id.
func (x *extraction) capabilityHistory(m *Model) (map[string][]changeRow, error) {
	status := map[string]string{}
	for i := range m.Entries {
		status[m.Entries[i].ID] = m.Entries[i].Status
	}
	arch := path.Join(x.dir, dirOr(x.o.Changes, defaultChanges), "archive")
	ents, err := os.ReadDir(filepath.Join(x.o.Root, filepath.FromSlash(arch)))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[string][]changeRow{}
	for _, e := range ents {
		var row changeRow
		switch name := e.Name(); {
		case !e.IsDir():
			continue
		case reSeedDate.MatchString(name):
			row.date = name[:len("2006-01-02")]
		case reArchived.MatchString(name):
			m := reArchived.FindStringSubmatch(name)
			row.date, row.id = m[1], m[2]
			if status[row.id] != "delivered" {
				continue
			}
		default:
			continue
		}
		specs, err := os.ReadDir(filepath.Join(x.o.Root, filepath.FromSlash(path.Join(arch, e.Name(), "specs"))))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, s := range specs {
			if s.IsDir() {
				out[s.Name()] = append(out[s.Name()], row)
			}
		}
	}
	for _, rows := range out {
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].date != rows[j].date {
				return rows[i].date < rows[j].date
			}
			return rows[i].id < rows[j].id
		})
	}
	return out, nil
}
