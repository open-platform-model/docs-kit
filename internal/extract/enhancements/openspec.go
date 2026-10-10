package enhancements

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	reChange   = regexp.MustCompile(`^(\d{4})-([a-z0-9]+(?:-[a-z0-9]+)*)$`)
	reArchived = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-(\d{4})-([a-z0-9]+(?:-[a-z0-9]+)*)$`)
	reSeedDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-seed-`)
	reCap      = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	reAccepted = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}) ([0-9a-f]{7,40})$`)
	reTask     = regexp.MustCompile(`^\s*[-*] \[([ xX])\] `)
)

// specConfig is the part of an openspec entry's config.yaml the section
// publishes. A stored `updated` is not read: the date comes from git.
type specConfig struct {
	ID           string   `json:"id"`
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	Status       string   `json:"status"`
	Category     string   `json:"category"`
	Breaking     *bool    `json:"breaking"`
	Affects      []string `json:"affects"`
	Created      string   `json:"created"`
	After        []string `json:"after"`
	Accepted     string   `json:"accepted"`
	SupersededBy *struct {
		IDs []string `json:"ids"`
	} `json:"superseded_by"`
}

// check refuses a config the section cannot publish as data.
func (c *specConfig) check() error {
	switch {
	case collapse(c.Title) == "":
		return fmt.Errorf("title is empty")
	case collapse(c.Summary) == "":
		return fmt.Errorf("summary is empty")
	case !reDate.MatchString(c.Created):
		return fmt.Errorf("created %q must be a date, YYYY-MM-DD", c.Created)
	case !reWord.MatchString(c.Status) || !reWord.MatchString(c.Category):
		return fmt.Errorf("status %q and category %q must be lower-case words", c.Status, c.Category)
	case c.Accepted != "" && !reAccepted.MatchString(c.Accepted):
		return fmt.Errorf(`accepted %q must read "<YYYY-MM-DD> <commit>"`, c.Accepted)
	}
	for _, a := range c.Affects {
		if !reRepo.MatchString(a) {
			return fmt.Errorf("affects: %q is not a repository name", a)
		}
	}
	ids := c.After
	if c.SupersededBy != nil {
		ids = append(append([]string{}, ids...), c.SupersededBy.IDs...)
	}
	for _, id := range ids {
		if !reID.MatchString(id) {
			return fmt.Errorf("after and superseded_by: %q is not an entry id", id)
		}
	}
	return nil
}

// openspecDirs lists the entries that are OpenSpec changes: the live
// <changes>/<id>-<slug>/ and the archived
// <changes>/archive/<date>-<id>-<slug>/. A seed change is no entry. A
// folder named any other way is refused.
func openspecDirs(root, dir, changes string) ([]entryDir, error) {
	parent := path.Join(dir, changes)
	live, err := scanChanges(root, parent, false)
	if err != nil {
		return nil, err
	}
	archived, err := scanChanges(root, path.Join(parent, "archive"), true)
	if err != nil {
		return nil, err
	}
	return append(live, archived...), nil
}

// scanChanges lists the change folders directly under parent.
func scanChanges(root, parent string, archived bool) ([]entryDir, error) {
	ents, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(parent)))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := checkDir(root, parent, parent); err != nil {
		return nil, err
	}
	var out []entryDir
	for _, e := range ents {
		d := path.Join(parent, e.Name())
		if e.Type()&fs.ModeSymlink != 0 {
			return nil, checkDir(root, d, "change "+d)
		}
		if !e.IsDir() || !archived && e.Name() == "archive" {
			continue
		}
		ed, isEntry, err := classifyChange(e.Name(), d, archived)
		if err != nil {
			return nil, err
		}
		if !isEntry {
			continue
		}
		if err := checkDir(root, d, "entry "+d); err != nil {
			return nil, err
		}
		ed.dir, ed.layout = d, LayoutOpenSpec
		out = append(out, ed)
	}
	return out, nil
}

// classifyChange reads a change folder's name: an entry (with its id,
// slug and, once archived, date), a seed (no entry), or a name it refuses.
func classifyChange(name, dir string, archived bool) (ed entryDir, isEntry bool, err error) {
	switch {
	case !archived && strings.HasPrefix(name, "seed-"), archived && reSeedDate.MatchString(name):
		return ed, false, nil
	case !archived && reChange.MatchString(name):
		m := reChange.FindStringSubmatch(name)
		return entryDir{id: m[1], slug: m[2]}, true, nil
	case archived && reArchived.MatchString(name):
		m := reArchived.FindStringSubmatch(name)
		return entryDir{id: m[2], slug: m[3], date: m[1], archived: true}, true, nil
	}
	form := "<id>-<slug>"
	if archived {
		form = "<date>-<id>-<slug>"
	}
	return ed, false, fmt.Errorf("change folder %s is not %s or a seed-<capability> change; an enhancement is a change named by its id and slug", dir, form)
}

// The pages of an openspec entry besides its README, in order. A page
// whose file is "" is built from the entry's delta specs.
var specDocs = []struct{ slug, title, description, file string }{
	{"proposal", "Proposal", "What changes, and why.", "proposal.md"},
	{"design", "Design", "How it works, and why this way.", "design.md"},
	{"requirements", "Requirements", "Exactly what must hold, by capability.", ""},
	{"tasks", "Tasks", "Which repository still has work.", "tasks.md"},
}

// specEntry reads one openspec entry: its config.yaml, README.md,
// proposal.md and design.md, and its tasks.md and delta specs when it has
// them.
func (x *extraction) specEntry(d entryDir) (Entry, error) {
	var c specConfig
	file, err := decodeConfig(x.o.Root, d, &c)
	if err != nil {
		return Entry{}, err
	}
	switch {
	case c.ID != d.id:
		return Entry{}, fmt.Errorf("entry %s: %s says id %q; an entry's id is its folder's id", d.id, file, c.ID)
	case c.Slug != d.slug:
		return Entry{}, fmt.Errorf("entry %s: %s says slug %q; an entry's slug is its folder's slug %q", d.id, file, c.Slug, d.slug)
	}
	if err := c.check(); err != nil {
		return Entry{}, fmt.Errorf("entry %s: %s: %w", d.id, file, err)
	}
	updated, err := x.updated(d, c.Created)
	if err != nil {
		return Entry{}, err
	}
	e := c.entry(d, updated)
	if err := x.specPages(d, &e); err != nil {
		return Entry{}, err
	}
	return e, nil
}

// entry is the header data of a checked config; the pages fill the rest.
func (c *specConfig) entry(d entryDir, updated string) Entry {
	e := Entry{
		ID: d.id, Layout: LayoutOpenSpec, Slug: c.Slug, Title: collapse(c.Title), Summary: collapse(c.Summary), Status: c.Status,
		Category: c.Category, Breaking: c.Breaking, Affects: nonNil(c.Affects), Created: c.Created, Updated: updated,
		Archived: d.archived, After: nonNil(c.After), SupersededIDs: []string{}, Page: d.id, Documents: []Document{},
		Capabilities: []string{}, Aliases: []Alias{},
	}
	if c.SupersededBy != nil {
		e.SupersededIDs = nonNil(c.SupersededBy.IDs)
	}
	if c.Accepted != "" {
		m := reAccepted.FindStringSubmatch(c.Accepted)
		e.Accepted = &Accepted{Date: m[1], Commit: m[2]}
	}
	return e
}

// specPages reads an openspec entry's files into its pages, documents,
// progress, capabilities and aliases.
func (x *extraction) specPages(d entryDir, e *Entry) error {
	id := d.id
	readme := path.Join(d.dir, "README.md")
	body, err := readFile(x.o.Root, readme)
	if err != nil {
		return fmt.Errorf("entry %s: %w", id, err)
	}
	x.pages[readme] = target{url: "/enhancements/" + id + "/", title: id + ": " + e.Title}
	x.out = append(x.out, Page{Path: id + "/_index.md", Title: id + ": " + e.Title, Description: e.Summary, Weight: atoi(id) + 1, Body: body, Source: readme})

	caps, err := x.deltaSpecs(d)
	if err != nil {
		return err
	}
	hasTasks, err := exists(x.o.Root, path.Join(d.dir, "tasks.md"))
	if err != nil {
		return err
	}
	have := map[string]bool{"proposal": true, "design": true, "requirements": len(caps) > 0, "tasks": hasTasks}
	for n, k := range specDocs {
		if !have[k.slug] {
			continue
		}
		page := id + "/" + k.slug
		pg := Page{Path: page + ".md", Title: id + ": " + k.title, Description: k.description, Type: "explanation", Weight: n + 1}
		doc := Document{Slug: k.slug, Title: k.title, Page: page}
		tgt := target{url: "/enhancements/" + page + "/", title: id + ": " + k.title}
		if k.file == "" {
			doc.File = path.Join(d.dir, "specs")
			x.pages[doc.File] = tgt
			for _, cs := range caps {
				x.pages[cs.file] = tgt
				part := Part{Heading: "## " + cs.name, Source: cs.file, Body: cs.text, Demote: true}
				if x.hasCapability(cs.name) {
					part.Intro = "Capability page: [`" + cs.name + "`](" + capabilityURL(cs.name) + ")."
				}
				pg.Parts = append(pg.Parts, part)
				pg.Inputs = append(pg.Inputs, cs.file)
			}
		} else {
			doc.File = path.Join(d.dir, k.file)
			raw, err := readFile(x.o.Root, doc.File)
			if err != nil {
				return fmt.Errorf("entry %s: %w", id, err)
			}
			x.pages[doc.File] = tgt
			pg.Body, pg.Source = raw, doc.File
			if k.slug == "tasks" {
				e.Progress = progress(raw)
			}
		}
		e.Documents = append(e.Documents, doc)
		x.out = append(x.out, pg)
	}
	for _, cs := range caps {
		e.Capabilities = append(e.Capabilities, cs.name)
	}
	// Only a migrated entry had old pages to redirect from; a new entry gets
	// none. docs-kit cannot read that from the id, so the migration marks it
	// with evidence/migration-map.md.
	migrated, err := regularFile(x.o.Root, path.Join(d.dir, "evidence", "migration-map.md"))
	if err != nil {
		return err
	}
	if migrated {
		e.Aliases = aliases(id, have)
	}
	return nil
}

// deltaSpec is one capability's delta spec file of an entry.
type deltaSpec struct{ name, file, text string }

// deltaSpecs reads an entry's specs/<capability>/spec.md files, in name
// order. A missing specs directory holds none.
func (x *extraction) deltaSpecs(d entryDir) ([]deltaSpec, error) {
	dir := path.Join(d.dir, "specs")
	ents, err := os.ReadDir(filepath.Join(x.o.Root, filepath.FromSlash(dir)))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := checkDir(x.o.Root, dir, dir); err != nil {
		return nil, err
	}
	var out []deltaSpec
	for _, e := range ents {
		sub := path.Join(dir, e.Name())
		if e.Type()&fs.ModeSymlink != 0 {
			return nil, checkDir(x.o.Root, sub, sub)
		}
		if !e.IsDir() {
			continue
		}
		if !reCap.MatchString(e.Name()) {
			return nil, fmt.Errorf("entry %s: %s is not a capability name; a capability is a lower-case word, with hyphens", d.id, sub)
		}
		file := path.Join(sub, "spec.md")
		text, err := readFile(x.o.Root, file)
		if err != nil {
			return nil, fmt.Errorf("entry %s: %w", d.id, err)
		}
		out = append(out, deltaSpec{name: e.Name(), file: file, text: text})
	}
	return out, nil
}

// aliases lists the old page of each retired slug and the page that
// replaces it: the entry's own page when the entry has no such page. The
// caller asks for it only for a migrated entry.
func aliases(id string, have map[string]bool) []Alias {
	var out []Alias
	for _, m := range []struct{ from, to string }{
		{"problem", "proposal"}, {"design", "design"}, {"decisions", "requirements"}, {"graduation", "tasks"},
		{"risks", "design"}, {"operational", "design"}, {"questions", "proposal"},
	} {
		if m.from == m.to {
			continue
		}
		to := id
		if have[m.to] {
			to = id + "/" + m.to
		}
		out = append(out, Alias{From: id + "/" + m.from, To: to})
	}
	return out
}

// progress counts the task lines of tasks.md outside code fences.
func progress(text string) Progress {
	var p Progress
	fence := ""
	for _, l := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if f, _, ok := fenceLine(l, fence); ok {
			if fence == "" {
				fence = f
			} else {
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		if m := reTask.FindStringSubmatch(l); m != nil {
			p.Total++
			if m[1] != " " {
				p.Done++
			}
		}
	}
	return p
}

// updated is the date of the last commit that touched the entry's folder,
// or the created date when no commit did (an entry not yet committed, in a
// local preview).
func (x *extraction) updated(d entryDir, created string) (string, error) {
	if x.o.LastCommit == nil {
		return created, nil
	}
	ts, err := x.o.LastCommit(d.dir)
	if err != nil {
		return "", fmt.Errorf("entry %s: %w", d.id, err)
	}
	if ts == "" {
		return created, nil
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return "", fmt.Errorf("entry %s: commit date %q: %w", d.id, ts, err)
	}
	return t.UTC().Format("2006-01-02"), nil
}

// assemble builds a page body from its parts: each part cleaned as its own
// file, its headings moved down when asked, under its heading.
func (x *extraction) assemble(parts []Part) (string, error) {
	var blocks []string
	for _, p := range parts {
		body, err := x.clean(p.Source, p.Body)
		if err != nil {
			return "", err
		}
		if p.Demote {
			if body, err = demoteHeadings(p.Source, body); err != nil {
				return "", err
			}
		}
		var b strings.Builder
		if p.Heading != "" {
			b.WriteString(p.Heading + "\n\n")
		}
		if p.Intro != "" {
			b.WriteString(p.Intro + "\n\n")
		}
		b.WriteString(strings.TrimRight(body, "\n"))
		blocks = append(blocks, b.String())
	}
	return strings.Join(blocks, "\n\n") + "\n", nil
}

// demoteHeadings moves every ATX heading outside code fences one level
// down, and refuses one that would pass level 6.
func demoteHeadings(file, text string) (string, error) {
	lines := strings.Split(text, "\n")
	fence := ""
	for i, l := range lines {
		if f, _, ok := fenceLine(l, fence); ok {
			if fence == "" {
				fence = f
			} else {
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		t := strings.TrimLeft(l, " ")
		if len(l)-len(t) > 3 || !strings.HasPrefix(t, "#") {
			continue
		}
		n := runLen(t, 0)
		if n > 6 || len(t) > n && t[n] != ' ' && t[n] != '\t' {
			continue // not a heading
		}
		if n == 6 {
			return "", fmt.Errorf("%s:%d: a level-6 heading cannot move down one level for the requirements page", file, i+1)
		}
		lines[i] = l[:len(l)-len(t)] + "#" + t
	}
	return strings.Join(lines, "\n"), nil
}

// exists reports whether a file path, relative to Root, is there; a
// symlink is there, and readFile refuses it.
func exists(root, file string) (bool, error) {
	_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(file)))
	switch {
	case os.IsNotExist(err):
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}

// regularFile reports whether file exists and is a regular file; a symbolic
// link is not one.
func regularFile(root, file string) (bool, error) {
	fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(file)))
	switch {
	case os.IsNotExist(err):
		return false, nil
	case err != nil:
		return false, err
	}
	return fi.Mode().IsRegular(), nil
}

// nonNil is l, or an empty list when l is nil, so the JSON holds [].
func nonNil(l []string) []string {
	if l == nil {
		return []string{}
	}
	return l
}

// hasCapability reports whether the main specs hold the capability.
func (x *extraction) hasCapability(name string) bool {
	for i := range x.caps {
		if x.caps[i].name == name {
			return true
		}
	}
	return false
}
