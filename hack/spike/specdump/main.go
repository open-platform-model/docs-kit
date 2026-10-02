// Command specdump loads a CUE catalog module and prints, for the named
// members, the structured spec walk the doc model records: every field under
// the member's spec key, depth first, with its path, type, presence, default
// and the outside definition it refers to. It is the spike for that walk.
//
// Usage:
//
//	go run ./hack/spike/specdump -module <catalog_opm>/opm container volumes backup
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/token"
)

const maxDepth = 12

func main() {
	dir := flag.String("module", "", "path to the catalog module root")
	flag.Parse()
	if *dir == "" || flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: specdump -module <dir> <member name>...")
		os.Exit(1)
	}
	if err := run(*dir, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "specdump:", err)
		os.Exit(2)
	}
}

func run(dir string, names []string) error {
	insts := load.Instances([]string{"."}, &load.Config{Dir: dir})
	if err := insts[0].Err; err != nil {
		return err
	}
	cat := cuecontext.New().BuildInstance(insts[0])
	if err := cat.Err(); err != nil {
		return err
	}
	for _, name := range names {
		m, err := find(cat, name)
		if err != nil {
			return err
		}
		if err := dump(m); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

// find returns the newest catalog map entry whose metadata.name is name.
func find(cat cue.Value, name string) (cue.Value, error) {
	var found cue.Value
	for _, kind := range []string{"#resources", "#traits", "#blueprints"} {
		it, err := cat.LookupPath(cue.ParsePath(kind)).Fields()
		if err != nil {
			return found, err
		}
		for it.Next() {
			n, _ := it.Value().LookupPath(cue.ParsePath("metadata.name")).String()
			if n == name {
				found = it.Value()
			}
		}
	}
	if !found.Exists() {
		return found, fmt.Errorf("no member named %q", name)
	}
	return found, nil
}

func dump(m cue.Value) error {
	fqn, _ := m.LookupPath(cue.ParsePath("metadata.fqn")).String()
	spec := m.LookupPath(cue.ParsePath("spec"))
	it, err := spec.Fields()
	if err != nil {
		return err
	}
	if !it.Next() {
		return fmt.Errorf("spec has no field")
	}
	key := it.Selector().Unquoted()
	pkg := memberPackage(m)
	fmt.Printf("== %s (spec key %s, package %s)\n", fqn, key, pkg)
	w := &walker{pkg: pkg, visited: map[string]bool{}}
	return w.walk(it.Value(), "", 0)
}

// memberPackage is the import path of the package that declares the
// member's definition, read from the reference the catalog map holds.
func memberPackage(m cue.Value) string {
	for _, c := range conjuncts(m) {
		root, p := c.ReferencePath()
		if len(p.Selectors()) > 0 && root.BuildInstance() != nil {
			return root.BuildInstance().ImportPath
		}
	}
	return ""
}

type walker struct {
	pkg     string
	visited map[string]bool
}

func (w *walker) walk(v cue.Value, path string, depth int) error {
	if depth > maxDepth {
		fmt.Printf("%s\t(depth cap)\n", path)
		return nil
	}
	for _, c := range children(v) {
		w.field(c.v, join(path, c.name), c.presence, depth)
	}
	// A pattern constraint, [string]: T.
	if pv := v.LookupPath(cue.MakePath(cue.AnyString)); pv.Exists() {
		w.field(pv, join(path, "[string]"), "regular", depth)
	}
	return nil
}

type child struct {
	name, presence string
	v              cue.Value
}

// children lists a struct's fields. When the evaluator cannot iterate the
// struct (a struct-level validator such as matchN, or an if-guard on a
// non-concrete field, leaves its kind open), the field names come from the
// value's syntax and each value from a lookup.
func children(v cue.Value) []child {
	var out []child
	if it, err := v.Fields(cue.Optional(true), cue.Definitions(false)); err == nil {
		for it.Next() {
			out = append(out, child{it.Selector().Unquoted(), presence(it.Selector()), it.Value()})
		}
		return out
	}
	lit := structLit(v.Syntax())
	if lit == nil {
		return nil
	}
	for _, e := range lit.Elts {
		f, ok := e.(*ast.Field)
		if !ok {
			continue
		}
		name, isIdent, err := ast.LabelName(f.Label)
		if err != nil || (isIdent && strings.HasPrefix(name, "#")) || strings.HasPrefix(name, "_") {
			continue
		}
		sel := cue.Str(name)
		pres := "regular"
		optional, required := f.Constraint == token.OPTION, f.Constraint == token.NOT
		if optional {
			sel, pres = sel.Optional(), "optional"
		}
		if required {
			sel, pres = sel.Required(), "required"
		}
		cv := v.LookupPath(cue.MakePath(sel))
		if !cv.Exists() {
			continue
		}
		out = append(out, child{name, pres, cv})
	}
	return out
}

// structLit finds the struct literal a value's syntax holds, inside a file
// or behind the _#def wrapper Syntax adds for a closed struct.
func structLit(n ast.Node) *ast.StructLit {
	switch x := n.(type) {
	case *ast.StructLit:
		return x
	case *ast.File:
		var lit *ast.StructLit
		for _, d := range x.Decls {
			switch d := d.(type) {
			case *ast.EmbedDecl:
				if l := structLit(d.Expr); l != nil {
					lit = l
				}
			case *ast.Field:
				if l := structLit(d.Value); l != nil {
					lit = l
				}
			}
		}
		if lit == nil {
			return &ast.StructLit{Elts: x.Decls}
		}
		return lit
	case *ast.BinaryExpr:
		if l := structLit(x.X); l != nil {
			return l
		}
		return structLit(x.Y)
	}
	return nil
}

func (w *walker) field(fv cue.Value, p, pres string, depth int) {
	ref := w.outsideRef(fv)
	def := "null"
	if d, ok := fv.Default(); ok && !fv.IsConcrete() {
		def = exprText(d.Syntax(cue.Final()))
	}
	fmt.Printf("%s\t%s\t%s\tdefault=%s\tref=%s\n", p, typeOf(fv), pres, def, refOrNull(ref))
	if ref != "" {
		return
	}
	switch {
	case isStruct(fv):
		key := defKey(fv)
		if key != "" {
			if w.visited[key] {
				fmt.Printf("%s\t(visited %s)\n", p, key)
				return
			}
			w.visited[key] = true
			defer delete(w.visited, key)
		}
		_ = w.walk(fv, p, depth+1)
	case fv.IncompleteKind() == cue.ListKind:
		if ev := fv.LookupPath(cue.MakePath(cue.AnyIndex)); ev.Exists() {
			w.field(ev, p+"[]", "regular", depth+1)
		}
	}
}

// isStruct reports a struct, including one whose kind a struct-level
// validator such as matchN widens to top.
func isStruct(v cue.Value) bool {
	k := v.IncompleteKind()
	if k == cue.StructKind {
		return true
	}
	if k&cue.StructKind == 0 && k != cue.BottomKind {
		return false
	}
	return len(children(v)) > 0
}

// outsideRef returns "<import path>.#Name" when a conjunct of v is a
// reference to a definition declared outside the member's package.
func (w *walker) outsideRef(v cue.Value) string {
	for _, c := range conjuncts(v) {
		root, p := c.ReferencePath()
		sels := p.Selectors()
		if len(sels) == 0 || root.BuildInstance() == nil {
			continue
		}
		last := sels[len(sels)-1]
		if !last.IsDefinition() {
			continue
		}
		ip := root.BuildInstance().ImportPath
		if ip != w.pkg {
			return ip + "." + p.String()
		}
	}
	return ""
}

func defKey(v cue.Value) string {
	for _, c := range conjuncts(v) {
		root, p := c.ReferencePath()
		if len(p.Selectors()) > 0 && root.BuildInstance() != nil {
			return root.BuildInstance().ImportPath + "." + p.String()
		}
	}
	return ""
}

func conjuncts(v cue.Value) []cue.Value {
	op, args := v.Expr()
	if op == cue.AndOp {
		return args
	}
	return []cue.Value{v}
}

func presence(sel cue.Selector) string {
	if t := sel.ConstraintType(); t == cue.OptionalConstraint {
		return "optional"
	} else if t == cue.RequiredConstraint {
		return "required"
	}
	return "regular"
}

func typeOf(v cue.Value) string {
	switch {
	case isStruct(v):
		return "struct"
	case v.IncompleteKind() == cue.ListKind:
		return "list"
	}
	if _, ok := v.Default(); ok && !v.IsConcrete() {
		// Keep the alternatives; Final would collapse to the default.
		return exprText(v.Syntax())
	}
	return exprText(v.Syntax(cue.Final()))
}

// exprText formats a value's syntax as one line, dropping the import
// declarations Syntax adds for builtins.
func exprText(n ast.Node) string {
	if f, ok := n.(*ast.File); ok {
		var decls []ast.Decl
		for _, d := range f.Decls {
			if _, imp := d.(*ast.ImportDecl); !imp {
				decls = append(decls, d)
			}
		}
		if len(decls) == 1 {
			if e, ok := decls[0].(*ast.EmbedDecl); ok {
				return fmtNode(e.Expr)
			}
		}
		return fmtNode(&ast.File{Decls: decls})
	}
	return fmtNode(n)
}

func fmtNode(n ast.Node) string {
	b, err := format.Node(n, format.Simplify())
	if err != nil {
		return "<" + err.Error() + ">"
	}
	return strings.Join(strings.Fields(string(b)), " ")
}

func refOrNull(s string) string {
	if s == "" {
		return "null"
	}
	return s
}

func join(a, b string) string {
	if a == "" {
		return b
	}
	return a + "." + b
}
