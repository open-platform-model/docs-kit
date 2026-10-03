package enhancements

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue/cuecontext"
	cueyaml "cuelang.org/go/encoding/yaml"
)

// Options configures one extraction.
type Options struct {
	Root string // the source tree, the repository's root
	// Dir is the entry root, relative to Root: "." or a directory holding
	// INDEX.md, GRAPH.md, the entries NNNN/ and archive/NNNN/.
	Dir    string
	Repo   string // "open-platform-model/enhancements", for GitHub links
	Commit string // the 40-hex commit built
	// Paths is every path of the commit's tree, relative to Root, with its
	// kind ("blob" or "tree"): a relative link resolves only to a path the
	// commit has.
	Paths map[string]string
	// Title and Description are the section page's front matter.
	Title       string
	Description string
}

// Page is one page of the section, its body cleaned and its links
// resolved, ready for its front matter.
type Page struct {
	Path        string // under content/, "0025/problem.md"
	Title       string
	Description string
	Type        string // "explanation" on a leaf page; "" on a section page
	Weight      int    // 0 for none
	Body        string
	Source      string // the repository file it came from, "0025/01-problem.md"
}

// Result is the data file's model and the section's pages, in path order.
type Result struct {
	Model *Model
	Pages []Page
}

var (
	reID       = regexp.MustCompile(`^\d{4}$`)
	reCrossRef = regexp.MustCompile(`^(\d{4}|legacy:\d{3})$`)
	reWord     = regexp.MustCompile(`^[a-z0-9]+([.-][a-z0-9]+)*$`)
	reDate     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	reDocFile  = regexp.MustCompile(`^0([1-9])-.*\.md$`)
)

// template is the entry id reserved for the template; it is not published.
const template = "0000"

// config is the part of an entry's config.yaml the section publishes.
type config struct {
	ID           string   `json:"id"`
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	Status       string   `json:"status"`
	Category     string   `json:"category"`
	Affects      []string `json:"affects"`
	Created      string   `json:"created"`
	Updated      string   `json:"updated"`
	DependsOn    []string `json:"depends_on"`
	Amends       []string `json:"amends"`
	Supersedes   []string `json:"supersedes"`
	Revives      []string `json:"revives"`
	SupersededBy *string  `json:"superseded_by"`
}

// entryDir is one entry's directory: its id and its path relative to Root.
type entryDir struct {
	id, dir string
}

// Extract reads every entry under the entry root and returns the data
// file's model and the section's pages.
func Extract(o Options) (*Result, error) {
	dir := path.Clean(filepath.ToSlash(o.Dir))
	if err := checkDir(o.Root, dir, "enhancements dir "+dir); err != nil {
		return nil, err
	}
	dirs, err := entryDirs(o.Root, dir)
	if err != nil {
		return nil, err
	}
	m := &Model{Schema: SchemaID, Repo: o.Repo, Entries: []Entry{}}
	x := &extraction{o: o, dir: dir, pages: map[string]target{}}
	for _, d := range dirs {
		e, err := x.entry(d)
		if err != nil {
			return nil, err
		}
		m.Entries = append(m.Entries, e)
	}
	if err := x.sectionPages(); err != nil {
		return nil, err
	}
	for i := range x.out {
		p := &x.out[i]
		if p.Body, err = x.clean(p.Source, p.Body); err != nil {
			return nil, err
		}
	}
	sort.Slice(x.out, func(i, j int) bool { return x.out[i].Path < x.out[j].Path })
	return &Result{Model: m, Pages: x.out}, nil
}

// extraction is one run's state: the pages read so far, and every source
// file that becomes a page, which relative links resolve to.
type extraction struct {
	o     Options
	dir   string            // the entry root, cleaned: "." or "sub"
	pages map[string]target // a source file, relative to Root -> its page
	out   []Page
}

// target is a page a relative link can resolve to.
type target struct {
	url   string // "/enhancements/0025/decisions/"
	title string // what a link whose text is the file name reads as
}

// rel is p (relative to the entry root) relative to Root.
func (x *extraction) rel(p string) string { return path.Join(x.dir, p) }

// entryDirs lists the entry directories, live ones and archive/NNNN/, by
// id, leaving out the template; an id both live and archived is refused.
func entryDirs(root, dir string) ([]entryDir, error) {
	var out []entryDir
	seen := map[string]string{}
	for _, parent := range []string{dir, path.Join(dir, "archive")} {
		ents, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(parent)))
		if os.IsNotExist(err) && parent != dir {
			continue
		}
		if err != nil {
			return nil, err
		}
		if parent != dir {
			if err := checkDir(root, parent, parent); err != nil {
				return nil, err
			}
		}
		for _, e := range ents {
			if !reID.MatchString(e.Name()) || e.Name() == template {
				continue
			}
			d := path.Join(parent, e.Name())
			if err := checkDir(root, d, "entry "+d); err != nil {
				return nil, err
			}
			if prev, dup := seen[e.Name()]; dup {
				return nil, fmt.Errorf("entry %s is both %s/ and %s/; an entry lives in one place", e.Name(), prev, d)
			}
			seen[e.Name()] = d
			out = append(out, entryDir{id: e.Name(), dir: d})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out, nil
}

// checkDir refuses a directory that is missing, not a directory, or
// reached through a symlink: every component of dir, from Root down, is
// checked. what names it in the error.
func checkDir(root, dir, what string) error {
	at := ""
	for _, c := range strings.Split(dir, "/") {
		if c == "." || c == "" {
			continue
		}
		at = path.Join(at, c)
		fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(at)))
		switch {
		case os.IsNotExist(err):
			return fmt.Errorf("%s does not exist", what)
		case err != nil:
			return err
		case fi.Mode()&fs.ModeSymlink != 0:
			return fmt.Errorf("%s is reached through the symlink %s; the enhancements source reads only the repository's own directories", what, at)
		case !fi.IsDir():
			return fmt.Errorf("%s is not a directory", what)
		}
	}
	return nil
}

// readFile reads a regular file, relative to Root, refusing a symlink.
func readFile(root, file string) (string, error) {
	full := filepath.Join(root, filepath.FromSlash(file))
	fi, err := os.Lstat(full)
	switch {
	case os.IsNotExist(err):
		return "", fmt.Errorf("%s is missing", file)
	case err != nil:
		return "", err
	case !fi.Mode().IsRegular():
		return "", fmt.Errorf("%s is not a regular file (a symlink or a directory); the enhancements source reads only regular files", file)
	}
	b, err := os.ReadFile(full)
	return string(b), err
}

// entry reads one entry: its config.yaml, README.md and seven documents.
func (x *extraction) entry(d entryDir) (Entry, error) {
	c, err := readConfig(x.o.Root, d)
	if err != nil {
		return Entry{}, err
	}
	readme := path.Join(d.dir, "README.md")
	body, err := readFile(x.o.Root, readme)
	if err != nil {
		return Entry{}, fmt.Errorf("entry %s: %w", d.id, err)
	}
	id, _ := strconv.Atoi(d.id)
	e := Entry{
		ID: d.id, Slug: c.Slug, Title: c.Title, Summary: c.Summary, Status: c.Status, Category: c.Category,
		Affects: c.Affects, Created: c.Created, Updated: c.Updated, Archived: path.Dir(d.dir) != x.dir,
		DependsOn: c.DependsOn, Amends: c.Amends, Supersedes: c.Supersedes, Revives: c.Revives, SupersededBy: c.SupersededBy,
		Page: d.id, Documents: []Document{},
	}
	x.pages[readme] = target{url: "/enhancements/" + d.id + "/", title: d.id + ": " + c.Title}
	x.out = append(x.out, Page{Path: d.id + "/_index.md", Title: d.id + ": " + c.Title, Description: c.Summary, Weight: id + 1, Body: body, Source: readme})
	files, err := docFiles(x.o.Root, d)
	if err != nil {
		return Entry{}, err
	}
	for n, k := range docKinds {
		file := files[n]
		body, err := readFile(x.o.Root, file)
		if err != nil {
			return Entry{}, fmt.Errorf("entry %s: %w", d.id, err)
		}
		page := d.id + "/" + k.slug
		e.Documents = append(e.Documents, Document{Slug: k.slug, Title: k.title, Page: page, File: file})
		x.pages[file] = target{url: "/enhancements/" + page + "/", title: d.id + ": " + k.title}
		x.out = append(x.out, Page{Path: page + ".md", Title: d.id + ": " + k.title, Description: k.description, Type: "explanation", Weight: n + 1, Body: body, Source: file})
	}
	return e, nil
}

// docFiles finds an entry's seven numbered documents: exactly one
// 0<n>-*.md for each n from 1 to 7, and no other numbered file.
func docFiles(root string, d entryDir) ([]string, error) {
	ents, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(d.dir)))
	if err != nil {
		return nil, err
	}
	byN := map[int][]string{}
	for _, f := range ents {
		m := reDocFile.FindStringSubmatch(f.Name())
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		byN[n] = append(byN[n], path.Join(d.dir, f.Name()))
	}
	out := make([]string, len(docKinds))
	for n := 1; n <= 9; n++ {
		files := byN[n]
		switch {
		case n > len(docKinds) && len(files) > 0:
			return nil, fmt.Errorf("entry %s holds %s; an entry has exactly the seven documents 01-*.md to 07-*.md", d.id, strings.Join(files, ", "))
		case n > len(docKinds):
		case len(files) == 0:
			return nil, fmt.Errorf("entry %s has no 0%d-*.md; an entry has exactly one file per numbered document (%s)", d.id, n, docKinds[n-1].slug)
		case len(files) > 1:
			return nil, fmt.Errorf("entry %s holds %s; an entry has exactly one 0%d-*.md (%s)", d.id, strings.Join(files, " and "), n, docKinds[n-1].slug)
		default:
			out[n-1] = files[0]
		}
	}
	return out, nil
}

// readConfig reads and checks an entry's config.yaml.
func readConfig(root string, d entryDir) (*config, error) {
	file := path.Join(d.dir, "config.yaml")
	raw, err := readFile(root, file)
	if err != nil {
		return nil, fmt.Errorf("entry %s: %w", d.id, err)
	}
	f, err := cueyaml.Extract(file, []byte(raw))
	if err != nil {
		return nil, fmt.Errorf("entry %s: %w", d.id, err)
	}
	v := cuecontext.New().BuildFile(f)
	if err := v.Err(); err != nil {
		return nil, fmt.Errorf("entry %s: %s: %w", d.id, file, err)
	}
	var c config
	if err := v.Decode(&c); err != nil {
		return nil, fmt.Errorf("entry %s: %s: %w", d.id, file, err)
	}
	if c.ID != d.id {
		return nil, fmt.Errorf("entry %s: %s says id %q; an entry's id is its directory's name", d.id, file, c.ID)
	}
	if err := c.check(); err != nil {
		return nil, fmt.Errorf("entry %s: %s: %w", d.id, file, err)
	}
	c.Title = collapse(c.Title)
	c.Summary = collapse(c.Summary)
	for _, l := range []*[]string{&c.Affects, &c.DependsOn, &c.Amends, &c.Supersedes, &c.Revives} {
		if *l == nil {
			*l = []string{}
		}
	}
	return &c, nil
}

// check refuses a config the section cannot publish as data: a missing
// title or summary, a date or relation not in its form.
func (c *config) check() error {
	switch {
	case collapse(c.Title) == "":
		return fmt.Errorf("title is empty")
	case collapse(c.Summary) == "":
		return fmt.Errorf("summary is empty")
	case !reDate.MatchString(c.Created) || !reDate.MatchString(c.Updated):
		return fmt.Errorf("created %q and updated %q must be dates, YYYY-MM-DD", c.Created, c.Updated)
	case !reWord.MatchString(c.Slug) || !reWord.MatchString(c.Status) || !reWord.MatchString(c.Category):
		return fmt.Errorf("slug %q, status %q and category %q must be lower-case words", c.Slug, c.Status, c.Category)
	}
	for _, a := range c.Affects {
		if !reWord.MatchString(a) {
			return fmt.Errorf("affects: %q is not a repository name", a)
		}
	}
	for _, l := range []struct {
		name string
		ids  []string
		re   *regexp.Regexp
	}{{"depends_on", c.DependsOn, reID}, {"amends", c.Amends, reID}, {"supersedes", c.Supersedes, reCrossRef}, {"revives", c.Revives, reCrossRef}} {
		for _, id := range l.ids {
			if !l.re.MatchString(id) {
				return fmt.Errorf("%s: %q is not an entry id", l.name, id)
			}
		}
	}
	if c.SupersededBy != nil && !reCrossRef.MatchString(*c.SupersededBy) {
		return fmt.Errorf("superseded_by: %q is not an entry id", *c.SupersededBy)
	}
	return nil
}

// collapse folds runs of whitespace into one space and trims the ends.
func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// sectionPages adds the section page (INDEX.md) and the graph (GRAPH.md).
func (x *extraction) sectionPages() error {
	for _, p := range []*struct {
		file string
		page Page
		t    target
	}{
		{"INDEX.md", Page{Path: "_index.md", Title: x.o.Title, Description: x.o.Description}, target{url: "/enhancements/", title: "the index"}},
		{"GRAPH.md", Page{Path: "graph.md", Title: graphTitle, Description: graphDescription, Type: "explanation", Weight: 1}, target{url: "/enhancements/graph/", title: "the relationship graph"}},
	} {
		file := x.rel(p.file)
		body, err := readFile(x.o.Root, file)
		if err != nil {
			return err
		}
		p.page.Body, p.page.Source = body, file
		x.pages[file] = p.t
		x.out = append(x.out, p.page)
	}
	return nil
}
