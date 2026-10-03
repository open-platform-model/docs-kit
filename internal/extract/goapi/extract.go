package goapi

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/doc"
	"go/doc/comment"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/open-platform-model/docs-kit/internal/doctext"
)

// Config is a go-api source's entry in docs-kit.cue (docs-kit C6, C20).
type Config struct {
	Module      string   `json:"module"`   // "./": the directory holding go.mod, repo-relative
	Root        string   `json:"root"`     // "./opm": page names are relative to it, module-relative
	Packages    []string `json:"packages"` // "./opm/...": module-relative patterns
	Section     string   `json:"section"`  // "reference/go-api/"
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Weight      *int     `json:"weight"`
}

// Options is one extraction.
type Options struct {
	Source  string // the source tree
	Config  Config
	Version string         // "1.0.0-beta.1" or "edge"
	Policy  doctext.Policy // what citations become
	// Lenient: the config came from outside the source tree (a backfill),
	// so a pattern matching no package warns instead of failing.
	Lenient bool
	Warn    func(string)
}

// Result is the model and, per package page, the files it was built from.
type Result struct {
	Model *Model
	// Files maps a package's page path under content/
	// ("reference/go-api/kernel.md") to its repo-relative files, by name.
	Files map[string][]string
}

// ctxt selects files as a linux/amd64 build without cgo would.
func ctxt() build.Context {
	c := build.Default
	c.GOOS, c.GOARCH, c.CgoEnabled = "linux", "amd64", false
	c.BuildTags = nil
	c.GOPATH, c.GOROOT = "", ""
	return c
}

// reDocsPage is a page name the bundle admits: lower-case kebab-case.
var reDocsPage = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// pkgDir is one package directory the patterns selected.
type pkgDir struct {
	rel   string   // module-relative slash path: "opm/kernel"
	files []string // base names, by name
}

// Extract reads the module's selected packages into the model.
func Extract(o Options) (*Result, error) {
	cfg := o.Config
	modDir := filepath.Join(o.Source, filepath.FromSlash(cfg.Module))
	if err := checkDir(modDir, cfg.Module, "module"); err != nil {
		return nil, err
	}
	gomod := filepath.Join(modDir, "go.mod")
	data, err := readRegular(gomod, path.Join(cfg.Module, "go.mod"))
	if err != nil {
		return nil, err
	}
	modPath := modfile.ModulePath(data)
	if modPath == "" {
		return nil, fmt.Errorf("%s holds no module line", path.Join(cfg.Module, "go.mod"))
	}
	root := path.Clean(cfg.Root)
	if root != "." {
		if err := checkDir(filepath.Join(modDir, filepath.FromSlash(root)), cfg.Root, "root"); err != nil {
			return nil, err
		}
	}
	dirs, err := selectDirs(modDir, cfg.Packages, o)
	if err != nil {
		return nil, err
	}
	x := &extraction{o: o, modPath: modPath, fset: token.NewFileSet(), byPath: map[string]*pkgState{}}
	res := &Result{Files: map[string][]string{}}
	pages := map[string]string{}
	for _, d := range dirs {
		page, err := pageName(root, d.rel)
		if err != nil {
			return nil, err
		}
		if prev, dup := pages[page]; dup {
			return nil, fmt.Errorf("packages %s and %s both have the page name %q; page names are directories relative to root with / as -", prev, d.rel, page)
		}
		pages[page] = d.rel
		st, err := x.load(modDir, d, page)
		if err != nil {
			return nil, err
		}
		if st == nil {
			continue // a command, which has no importable API
		}
		x.pkgs = append(x.pkgs, st)
		x.byPath[st.importPath] = st
		res.Files[cfg.Section+page+".md"] = st.files
	}
	m, err := x.model()
	if err != nil {
		return nil, err
	}
	res.Model = m
	return res, nil
}

// pageName is a package's page: its directory relative to root, "/" as
// "-". A package at root, or outside it, has none.
func pageName(root, rel string) (string, error) {
	var sub string
	switch {
	case root == ".":
		sub = rel
	case rel == root:
		sub = ""
	case strings.HasPrefix(rel, root+"/"):
		sub = strings.TrimPrefix(rel, root+"/")
	default:
		return "", fmt.Errorf("package ./%s lies outside root %s; give root a directory that holds every selected package", rel, "./"+root)
	}
	if sub == "" || sub == "." {
		return "", fmt.Errorf("package ./%s is root itself, so it has no page name; give root its parent directory", rel)
	}
	page := strings.ReplaceAll(sub, "/", "-")
	if !reDocsPage.MatchString(page) {
		return "", fmt.Errorf("package ./%s has the page name %q, which is not lower-case kebab-case", rel, page)
	}
	return page, nil
}

// selectDirs walks the module and returns every package directory a
// pattern matches, by path: Go files selected for linux/amd64, no test
// file, no directory with an internal element.
func selectDirs(modDir string, patterns []string, o Options) ([]pkgDir, error) {
	w := &walker{modDir: modDir, ps: make(matchers, len(patterns)), hits: make([]int, len(patterns)), ctx: ctxt()}
	for i, p := range patterns {
		w.ps[i] = patternRE(p)
	}
	if err := filepath.WalkDir(modDir, w.visit); err != nil {
		return nil, err
	}
	for i, n := range w.hits {
		if n > 0 {
			continue
		}
		msg := fmt.Sprintf("packages %s matches no package under %s", patterns[i], o.Config.Module)
		if !o.Lenient {
			return nil, errors.New(msg)
		}
		if o.Warn != nil {
			o.Warn(msg)
		}
	}
	slices.SortFunc(w.out, func(a, b pkgDir) int { return strings.Compare(a.rel, b.rel) })
	return w.out, nil
}

// walker selects package directories during the module walk.
type walker struct {
	modDir string
	ps     matchers
	hits   []int // per pattern, the packages it matched
	ctx    build.Context
	out    []pkgDir
}

func (w *walker) visit(p string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	rel, _ := filepath.Rel(w.modDir, p)
	rel = filepath.ToSlash(rel)
	if d.Type()&fs.ModeSymlink != 0 {
		// A linked .go file is refused with its package; a linked
		// directory a pattern selects is refused here.
		if fi, err := os.Stat(p); err == nil && fi.IsDir() && len(w.ps.which(rel)) > 0 {
			return fmt.Errorf("./%s is a symbolic link; a go-api source reads only regular files and directories", rel)
		}
		return nil
	}
	if !d.IsDir() {
		return nil
	}
	if rel != "." && notInModule(p, d.Name()) {
		return filepath.SkipDir
	}
	which := w.ps.which(rel)
	if len(which) == 0 {
		return nil
	}
	files, err := goFiles(&w.ctx, p, rel)
	if err != nil || len(files) == 0 {
		return err
	}
	for _, i := range which {
		w.hits[i]++
	}
	w.out = append(w.out, pkgDir{rel: rel, files: files})
	return nil
}

// notInModule reports a directory the walk leaves out with everything
// under it: an internal package's, which is no public API, and what the
// go command does not count as the module's (testdata, vendor, a name
// starting with "." or "_", a nested module).
func notInModule(dir, base string) bool {
	if base == "internal" || base == "testdata" || base == "vendor" || strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_") {
		return true
	}
	_, err := os.Lstat(filepath.Join(dir, "go.mod"))
	return err == nil
}

// matchers are the source's package patterns.
type matchers []*regexp.Regexp

// which lists the patterns that match a module-relative path.
func (ms matchers) which(rel string) []int {
	var out []int
	for i, m := range ms {
		if m.MatchString(rel) {
			out = append(out, i)
		}
	}
	return out
}

// patternRE turns a module-relative package pattern into a regexp over
// module-relative slash paths: "..." matches any string, and a trailing
// "/..." also matches the directory itself, as the go command does.
func patternRE(p string) *regexp.Regexp {
	p = strings.TrimPrefix(path.Clean(p), "./")
	if p == "." {
		p = ""
	}
	re := regexp.QuoteMeta(p)
	re = strings.ReplaceAll(re, `\.\.\.`, `.*`)
	switch {
	case p == "...":
		re = `.*`
	case strings.HasSuffix(re, `/.*`):
		re = strings.TrimSuffix(re, `/.*`) + `(/.*)?`
	case p == "":
		re = `\.`
	}
	return regexp.MustCompile(`^` + re + `$`)
}

// goFiles lists a directory's Go files a linux/amd64 build without cgo
// selects, test files left out; a symbolic link is refused.
func goFiles(ctx *build.Context, dir, rel string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if !e.Type().IsRegular() {
			return nil, fmt.Errorf("./%s is not a regular file (%s); a go-api source reads only regular files", path.Join(rel, name), e.Type())
		}
		ok, err := ctx.MatchFile(dir, name)
		if err != nil {
			return nil, fmt.Errorf("./%s: %w", path.Join(rel, name), err)
		}
		if ok {
			out = append(out, name)
		}
	}
	return out, nil
}

func checkDir(dir, rel, option string) error {
	fi, err := os.Lstat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("%s %s does not exist", option, rel)
	case err != nil:
		return err
	case fi.Mode()&fs.ModeSymlink != 0:
		return fmt.Errorf("%s %s is a symbolic link; name the directory itself", option, rel)
	case !fi.IsDir():
		return fmt.Errorf("%s %s is not a directory", option, rel)
	}
	return nil
}

func readRegular(file, rel string) ([]byte, error) {
	fi, err := os.Lstat(file)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file (%s); a go-api source reads only regular files", rel, fi.Mode().Type())
	}
	return os.ReadFile(file)
}

// pkgState is one package read: go/doc's view, its parsed comments and,
// once assigned, its symbols' anchors.
type pkgState struct {
	importPath string
	rel        string // module-relative: "opm/kernel"
	page       string
	files      []string // repo-relative
	pkg        *doc.Package
	parser     *comment.Parser
	comments   map[*ast.CommentGroup]bool // the comment groups a declaration may print
	anchors    map[string]string          // "Kernel", "Kernel.Render" -> anchor
}

type extraction struct {
	o       Options
	modPath string
	fset    *token.FileSet
	pkgs    []*pkgState
	byPath  map[string]*pkgState
}

// load parses one package directory; a main package yields nil.
func (x *extraction) load(modDir string, d pkgDir, page string) (*pkgState, error) {
	var files []*ast.File
	st := &pkgState{rel: d.rel, page: page, comments: map[*ast.CommentGroup]bool{}}
	srcRel := func(name string) string {
		r, _ := filepath.Rel(x.o.Source, filepath.Join(modDir, filepath.FromSlash(d.rel), name))
		return filepath.ToSlash(r)
	}
	for _, name := range d.files {
		rel := srcRel(name)
		src, err := readRegular(filepath.Join(modDir, filepath.FromSlash(d.rel), name), rel)
		if err != nil {
			return nil, err
		}
		f, err := parser.ParseFile(x.fset, rel, src, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		if len(files) > 0 && f.Name.Name != files[0].Name.Name {
			return nil, fmt.Errorf("./%s holds packages %s and %s; a directory holds one package", d.rel, files[0].Name.Name, f.Name.Name)
		}
		files = append(files, f)
		st.files = append(st.files, rel)
	}
	if files[0].Name.Name == "main" {
		return nil, nil
	}
	st.importPath = x.modPath
	if d.rel != "." {
		st.importPath += "/" + d.rel
	}
	p, err := doc.NewFromFiles(x.fset, files, st.importPath)
	if err != nil {
		return nil, fmt.Errorf("./%s: %w", d.rel, err)
	}
	st.pkg = p
	st.parser = p.Parser()
	own := st.parser.LookupPackage
	st.parser.LookupPackage = func(name string) (string, bool) {
		if ip, ok := own(name); ok {
			return ip, true
		}
		return x.modulePackage(name)
	}
	return st, nil
}

// modulePackage resolves a package name no import of the package names to
// the one documented package of that name.
func (x *extraction) modulePackage(name string) (string, bool) {
	found := ""
	for _, p := range x.pkgs {
		if p.pkg.Name == name {
			if found != "" {
				return "", false
			}
			found = p.importPath
		}
	}
	return found, found != ""
}

// Heading levels: the package doc's headings start at ###; a symbol under
// ### has its doc's headings at ####, one under #### at #####.
const (
	levelPackageDoc = 3
	levelSymbolDoc  = 4
	levelMemberDoc  = 5
)

// model builds the model in two passes: the first prints every doc to
// learn the page's headings and assigns the anchors, the second prints
// again with doc links resolved to those anchors.
func (x *extraction) model() (*Model, error) {
	cfg := x.o.Config
	m := &Model{
		Schema: SchemaID, ModulePath: x.modPath, Version: x.o.Version,
		Section: cfg.Section, Title: cfg.Title, Description: cfg.Description, Weight: cfg.Weight,
	}
	for _, st := range x.pkgs {
		p, err := x.pkg(st, func(*comment.DocLink) string { return "" })
		if err != nil {
			return nil, err
		}
		st.anchors = assignAnchors(p)
	}
	for _, st := range x.pkgs {
		p, err := x.pkg(st, x.linkURL(st))
		if err != nil {
			return nil, err
		}
		m.Packages = append(m.Packages, *p)
	}
	slices.SortFunc(m.Packages, func(a, b Package) int { return strings.Compare(a.ImportPath, b.ImportPath) })
	return m, nil
}

// pkg builds one package's entry, its anchors taken from st when set.
func (x *extraction) pkg(st *pkgState, url func(*comment.DocLink) string) (*Package, error) {
	p := st.pkg
	pr := &mdPrinter{policy: x.o.Policy, docURL: url, where: "package ./" + st.rel}
	docText := dropWhyLines(p.Doc)
	pkgDoc, err := pr.markdown(st.parser.Parse(docText), levelPackageDoc)
	if err != nil {
		return nil, err
	}
	out := &Package{
		ImportPath: st.importPath, Name: p.Name, Page: st.page,
		Synopsis: doctext.Clean(p.Synopsis(docText)), Doc: pkgDoc, Files: st.files,
	}
	for _, f := range p.Funcs {
		fn, err := x.fn(st, f, "", levelSymbolDoc, url)
		if err != nil {
			return nil, err
		}
		out.Funcs = append(out.Funcs, fn)
	}
	if out.Consts, err = x.values(st, p.Consts, levelSymbolDoc, url); err != nil {
		return nil, err
	}
	if out.Vars, err = x.values(st, p.Vars, levelSymbolDoc, url); err != nil {
		return nil, err
	}
	for _, t := range p.Types {
		ty := Type{Name: t.Name, Anchor: st.anchors[t.Name]}
		if ty.Decl, err = x.decl(st, t.Decl); err != nil {
			return nil, err
		}
		if ty.Doc, err = x.doc(st, t.Doc, t.Name, levelSymbolDoc, url); err != nil {
			return nil, err
		}
		if ty.Consts, err = x.values(st, t.Consts, levelMemberDoc, url); err != nil {
			return nil, err
		}
		if ty.Vars, err = x.values(st, t.Vars, levelMemberDoc, url); err != nil {
			return nil, err
		}
		for _, f := range t.Funcs {
			fn, err := x.fn(st, f, "", levelMemberDoc, url)
			if err != nil {
				return nil, err
			}
			ty.Funcs = append(ty.Funcs, fn)
		}
		for _, f := range t.Methods {
			fn, err := x.fn(st, f, t.Name, levelMemberDoc, url)
			if err != nil {
				return nil, err
			}
			ty.Methods = append(ty.Methods, fn)
		}
		out.Types = append(out.Types, ty)
	}
	return out, nil
}

func (x *extraction) fn(st *pkgState, f *doc.Func, typ string, level int, url func(*comment.DocLink) string) (Func, error) {
	key := f.Name
	var recv *string
	if typ != "" {
		key = typ + "." + f.Name
		r := f.Recv
		recv = &r
	}
	decl, err := x.decl(st, f.Decl)
	if err != nil {
		return Func{}, err
	}
	d, err := x.doc(st, f.Doc, key, level, url)
	if err != nil {
		return Func{}, err
	}
	return Func{Name: f.Name, Recv: recv, Anchor: st.anchors[key], Decl: decl, Doc: d}, nil
}

func (x *extraction) values(st *pkgState, vs []*doc.Value, level int, url func(*comment.DocLink) string) ([]Value, error) {
	var out []Value
	for _, v := range vs {
		if len(v.Names) == 0 {
			continue
		}
		decl, err := x.decl(st, v.Decl)
		if err != nil {
			return nil, err
		}
		d, err := x.doc(st, v.Doc, v.Names[0], level, url)
		if err != nil {
			return nil, err
		}
		out = append(out, Value{Names: v.Names, Anchor: st.anchors[v.Names[0]], Decl: decl, Doc: d})
	}
	return out, nil
}

func (x *extraction) doc(st *pkgState, text, name string, level int, url func(*comment.DocLink) string) (string, error) {
	pr := &mdPrinter{policy: x.o.Policy, docURL: url, where: st.pkg.Name + "." + name + " (./" + st.rel + ")"}
	return pr.markdown(st.parser.Parse(dropWhyLines(text)), level)
}

// dropWhyLines removes the rationale lines of a doc comment.
func dropWhyLines(s string) string {
	lines := strings.Split(s, "\n")
	kept := lines[:0]
	for _, l := range lines {
		if !doctext.IsWhyLine(l) {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "\n")
}

// linkURL resolves a doc link: a package or symbol documented in this
// bundle links its page and anchor; anything else links pkg.go.dev.
func (x *extraction) linkURL(from *pkgState) func(*comment.DocLink) string {
	return func(l *comment.DocLink) string {
		ip := l.ImportPath
		if ip == "" {
			ip = from.importPath
		}
		key := l.Name
		if l.Recv != "" {
			key = l.Recv + "." + l.Name
		}
		if st, ok := x.byPath[ip]; ok {
			page := "/docs/" + x.o.Config.Section + st.page + "/"
			if l.Name == "" {
				return page
			}
			if a, ok := st.anchors[key]; ok {
				return page + "#" + a
			}
		}
		u := "https://pkg.go.dev/" + ip
		if key != "" {
			u += "#" + key
		}
		return u
	}
}

// decl prints a declaration as gofmt does, without its doc comment, with
// the comments of its fields and specs (rationale and banner groups
// dropped, citations removed). go/doc owns the parsed tree and has
// already read every doc, so the comments are removed in place; doing so
// twice changes nothing.
func (x *extraction) decl(st *pkgState, n ast.Node) (string, error) {
	switch d := n.(type) {
	case *ast.FuncDecl:
		d.Doc = nil
	case *ast.GenDecl:
		d.Doc = nil
		if !d.Lparen.IsValid() {
			// A lone spec: its own doc is the symbol's doc.
			for _, sp := range d.Specs {
				switch sp := sp.(type) {
				case *ast.TypeSpec:
					sp.Doc = nil
				case *ast.ValueSpec:
					sp.Doc = nil
				}
			}
		}
	default:
		return "", fmt.Errorf("./%s: unexpected declaration %T", st.rel, n)
	}
	// The printer prints the comments a node holds, and only in that mode
	// the note that a struct's unexported fields are filtered.
	drop := func(g **ast.CommentGroup) {
		if *g != nil && maintainerGroup(*g) {
			*g = nil
		}
	}
	ast.Inspect(n, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Field:
			drop(&n.Doc)
			drop(&n.Comment)
		case *ast.ValueSpec:
			drop(&n.Doc)
			drop(&n.Comment)
		case *ast.TypeSpec:
			drop(&n.Doc)
			drop(&n.Comment)
		}
		return true
	})
	var b strings.Builder
	if err := format.Node(&b, x.fset, n); err != nil {
		return "", fmt.Errorf("./%s: printing a declaration: %w", st.rel, err)
	}
	return doctext.CleanCode(strings.TrimRight(b.String(), "\n")), nil
}

// maintainerGroup reports a comment group for maintainers: its first line
// starts with WHY or is a //// banner.
func maintainerGroup(g *ast.CommentGroup) bool {
	if len(g.List) == 0 {
		return false
	}
	t := strings.TrimSpace(strings.TrimPrefix(g.List[0].Text, "//"))
	return strings.HasPrefix(t, "WHY") || strings.HasPrefix(t, "//")
}

// assignAnchors walks a package page's headings in the order the renderer
// writes them and assigns each symbol its anchor, as Hugo makes them
// unique.
func assignAnchors(p *Package) map[string]string {
	var a Anchors
	out := map[string]string{}
	docs := func(md string) {
		for _, h := range Headings(md) {
			a.Next(HeadingAnchor(h))
		}
	}
	sym := func(key, heading, md string) {
		out[key] = a.Next(Anchor(heading))
		docs(md)
	}
	docs(p.Doc)
	for _, s := range PageSections(p) {
		a.Next(Anchor(s.Heading))
		for _, e := range s.Entries {
			sym(e.Key, e.Heading, e.Doc)
		}
	}
	return out
}

// Section is one "##" section of a package page.
type Section struct {
	Heading string // "Constants"
	Entries []Entry
}

// Entry is one symbol of a page section, in page order: a type's own
// constants, variables, constructors and methods follow it, one level
// deeper.
type Entry struct {
	Key     string // what a doc link names: "Kernel", "Kernel.Render"
	Heading string // "Kernel.Render"; a value group is headed by its first name
	Level   int    // 3, or 4 under a type
	Anchor  string
	Decl    string
	Doc     string
}

// PageSections lists a package page's sections and their entries in page
// order; a section without entries is left out. The extractor assigns
// anchors in this order and the renderer writes headings in it.
func PageSections(p *Package) []Section {
	val := func(v Value, level int) Entry {
		return Entry{Key: v.Names[0], Heading: v.Names[0], Level: level, Anchor: v.Anchor, Decl: v.Decl, Doc: v.Doc}
	}
	var out []Section
	add := func(heading string, es []Entry) {
		if len(es) > 0 {
			out = append(out, Section{Heading: heading, Entries: es})
		}
	}
	es := make([]Entry, 0, len(p.Consts))
	for _, v := range p.Consts {
		es = append(es, val(v, 3))
	}
	add("Constants", es)
	es = make([]Entry, 0, len(p.Vars))
	for _, v := range p.Vars {
		es = append(es, val(v, 3))
	}
	add("Variables", es)
	es = make([]Entry, 0, len(p.Funcs))
	for _, f := range p.Funcs {
		es = append(es, Entry{Key: f.Name, Heading: f.Name, Level: 3, Anchor: f.Anchor, Decl: f.Decl, Doc: f.Doc})
	}
	add("Functions", es)
	es = nil
	for i := range p.Types {
		t := &p.Types[i]
		es = append(es, Entry{Key: t.Name, Heading: t.Name, Level: 3, Anchor: t.Anchor, Decl: t.Decl, Doc: t.Doc})
		for _, v := range t.Consts {
			es = append(es, val(v, 4))
		}
		for _, v := range t.Vars {
			es = append(es, val(v, 4))
		}
		for _, f := range t.Funcs {
			es = append(es, Entry{Key: f.Name, Heading: f.Name, Level: 4, Anchor: f.Anchor, Decl: f.Decl, Doc: f.Doc})
		}
		for _, f := range t.Methods {
			key := t.Name + "." + f.Name
			es = append(es, Entry{Key: key, Heading: key, Level: 4, Anchor: f.Anchor, Decl: f.Decl, Doc: f.Doc})
		}
	}
	add("Types", es)
	return out
}
