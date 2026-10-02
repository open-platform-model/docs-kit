package cuecatalog

import (
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/token"

	"github.com/open-platform-model/docs-kit/internal/doctext"
)

// maxDepth caps the structured spec walk.
const maxDepth = 12

// Field presence.
const (
	presenceRegular  = "regular"
	presenceOptional = "optional"
	presenceRequired = "required"
)

// specFields walks the evaluated spec value depth first and lists every
// field: its dot path ("[]" for a list element, "[string]" for a pattern
// constraint), type, presence, default, cleaned doc and, for a definition
// outside the member's package, the reference where the walk stops.
func specFields(spec cue.Value, memberPkg string) []Field {
	w := &walker{pkg: stripMajor(memberPkg), visited: map[string]bool{}}
	if key := defKey(spec); key != "" {
		w.visited[key] = true
	}
	switch {
	case isStruct(spec):
		w.walk(spec, "", 0)
	case spec.IncompleteKind() == cue.ListKind:
		if ev := spec.LookupPath(cue.MakePath(cue.AnyIndex)); ev.Exists() {
			w.field(ev, "[]", presenceRegular, 0)
		}
	}
	return w.out
}

type walker struct {
	pkg     string
	visited map[string]bool
	out     []Field
}

func (w *walker) walk(v cue.Value, path string, depth int) {
	if depth > maxDepth {
		return
	}
	for _, c := range children(v) {
		w.field(c.v, join(path, c.name), c.presence, depth)
	}
	if pv := v.LookupPath(cue.MakePath(cue.AnyString)); pv.Exists() {
		w.field(pv, join(path, "[string]"), presenceRegular, depth)
	}
}

func (w *walker) field(fv cue.Value, p, presence string, depth int) {
	f := Field{Path: p, Type: typeOf(fv), Presence: presence, Doc: fieldDoc(fv)}
	if d, ok := fv.Default(); ok && !fv.IsConcrete() {
		s := exprText(d.Syntax(cue.Final()))
		f.Default = &s
	}
	if r := w.outsideRef(fv); r != "" {
		f.Ref = &r
		w.out = append(w.out, f)
		return
	}
	w.out = append(w.out, f)
	switch {
	case isStruct(fv):
		key := defKey(fv)
		if key != "" {
			if w.visited[key] {
				return
			}
			w.visited[key] = true
			defer delete(w.visited, key)
		}
		w.walk(fv, p, depth+1)
	case fv.IncompleteKind() == cue.ListKind:
		if ev := fv.LookupPath(cue.MakePath(cue.AnyIndex)); ev.Exists() && depth < maxDepth {
			w.field(ev, p+"[]", presenceRegular, depth+1)
		}
	}
}

// fieldDoc is a field's doc comment, maintainer comments dropped and
// citations stripped, paragraphs separated by a blank line.
func fieldDoc(v cue.Value) string {
	paras := doctext.CleanParagraphs(doctext.Paragraphs(doctext.GroupsText(v.Doc())))
	return strings.Join(paras, "\n\n")
}

type child struct {
	name, presence string
	v              cue.Value
}

// children lists a struct's fields. When the evaluator cannot iterate the
// struct (a struct-level validator such as matchN, or an if-guard on a
// non-concrete field, leaves its kind open), the names come from the
// value's syntax, in declaration order, and each value from a lookup.
func children(v cue.Value) []child {
	var out []child
	if it, err := v.Fields(cue.Optional(true), cue.Definitions(false)); err == nil {
		for it.Next() {
			out = append(out, child{it.Selector().Unquoted(), presenceOf(it.Selector()), it.Value()})
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
		sel, presence := cue.Str(name), presenceRegular
		optional, required := f.Constraint == token.OPTION, f.Constraint == token.NOT
		if optional {
			sel, presence = sel.Optional(), presenceOptional
		}
		if required {
			sel, presence = sel.Required(), presenceRequired
		}
		if cv := v.LookupPath(cue.MakePath(sel)); cv.Exists() {
			out = append(out, child{name, presence, cv})
		}
	}
	return out
}

// structLit finds the struct literal a value's syntax holds, inside a file
// or behind the definition wrapper Syntax adds for a closed struct.
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

// isStruct reports a struct, including one whose kind a struct-level
// validator or an if-guard leaves open.
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

// outsideRef returns "<import path>.<definition>" when a conjunct of v
// references a definition declared outside the member's package.
func (w *walker) outsideRef(v cue.Value) string {
	for _, c := range conjuncts(v) {
		root, p := c.ReferencePath()
		sels := p.Selectors()
		if len(sels) == 0 || root.BuildInstance() == nil || !sels[len(sels)-1].IsDefinition() {
			continue
		}
		ip := root.BuildInstance().ImportPath
		if stripMajor(ip) != w.pkg {
			return stripMajor(ip) + "." + p.String()
		}
	}
	return ""
}

// defKey names the definition a value refers to, for the visited set.
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

func presenceOf(sel cue.Selector) string {
	if t := sel.ConstraintType(); t == cue.OptionalConstraint {
		return presenceOptional
	} else if t == cue.RequiredConstraint {
		return presenceRequired
	}
	return presenceRegular
}

// typeOf is "struct", "list", or a scalar's evaluated constraint on one
// line; a value with a default keeps its alternatives.
func typeOf(v cue.Value) string {
	switch {
	case isStruct(v):
		return "struct"
	case v.IncompleteKind() == cue.ListKind:
		return "list"
	}
	if _, ok := v.Default(); ok && !v.IsConcrete() {
		return exprText(v.Syntax())
	}
	return exprText(v.Syntax(cue.Final()))
}

// exprText formats a value's syntax on one line, dropping the import
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
				return oneLine(e.Expr)
			}
		}
		return oneLine(&ast.File{Decls: decls})
	}
	return oneLine(n)
}

func oneLine(n ast.Node) string {
	b, err := format.Node(n, format.Simplify())
	if err != nil {
		return "_"
	}
	return strings.Join(strings.Fields(string(b)), " ")
}

func join(a, b string) string {
	if a == "" {
		return b
	}
	return a + "." + b
}
