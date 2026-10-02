package cuecatalog

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"

	"github.com/open-platform-model/docs-kit/internal/doctext"
)

// ref is a reference to a definition: a package-local `#Name`, or an
// imported `alias.#Name`.
type ref struct {
	pkg   string // import path as written (the defining package for a local ref)
	name  string
	alias string // the import alias, empty for a local reference
}

func (r ref) key() string { return stripMajor(r.pkg) + "." + r.name }

// display is the reference as an author writes it.
func (r ref) display() string {
	if r.alias == "" {
		return r.name
	}
	return r.alias + "." + r.name
}

// vendoredPrefix marks the vendored Kubernetes types: upstream API shapes,
// named rather than printed in full.
const vendoredPrefix = "/schemas/kubernetes/"

// fieldIn finds a field of the struct literal a definition unifies with its
// core kind (`c.#Trait & {...}`), or of a plain struct literal.
func fieldIn(e ast.Expr, name string) *ast.Field {
	switch x := e.(type) {
	case *ast.BinaryExpr:
		if f := fieldIn(x.X, name); f != nil {
			return f
		}
		return fieldIn(x.Y, name)
	case *ast.StructLit:
		for _, el := range x.Elts {
			if f, ok := el.(*ast.Field); ok {
				if n, _, err := ast.LabelName(f.Label); err == nil && n == name {
					return f
				}
			}
		}
	case *ast.ParenExpr:
		return fieldIn(x.X, name)
	}
	return nil
}

// authoredLabel returns one matchLabels value as the definition writes it,
// or "" when the definition does not write it.
func authoredLabel(d *defSrc, key string) string {
	ml := fieldIn(d.field.Value, "matchLabels")
	if ml == nil {
		return ""
	}
	f := fieldIn(ml.Value, key)
	if f == nil {
		return ""
	}
	s, err := formatNode(f.Value)
	if err != nil {
		return ""
	}
	return s
}

// refsIn lists the definition references in n, in source order and without
// repeats. Field labels are not references; a local name counts only when
// the package declares it at top level.
func refsIn(n ast.Node, ctx *defSrc) []ref {
	var out []ref
	seen := map[string]bool{}
	add := func(r ref) {
		if !seen[r.key()] {
			seen[r.key()] = true
			out = append(out, r)
		}
	}
	var before func(ast.Node) bool
	before = func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Field:
			if x.Value != nil {
				ast.Walk(x.Value, before, nil)
			}
			return false
		case *ast.SelectorExpr:
			if id, ok := x.X.(*ast.Ident); ok {
				if path, ok := ctx.imports[id.Name]; ok {
					if sel, _, err := ast.LabelName(x.Sel); err == nil && isDefName(sel) {
						add(ref{pkg: path, name: sel, alias: id.Name})
					}
					return false
				}
			}
		case *ast.Ident:
			if isDefName(x.Name) {
				if _, ok := ctx.pkg.defs[x.Name]; ok {
					add(ref{pkg: ctx.pkg.importPath, name: x.Name})
				}
			}
		}
		return true
	}
	ast.Walk(n, before, nil)
	return out
}

// specRoots maps every definition a member's spec field references directly
// to that member. A spec root of another member is linked, not repeated.
func specRoots(mod *module) map[string]*member {
	roots := map[string]*member{}
	for _, m := range mod.members {
		sf := fieldIn(m.def.field.Value, "spec")
		if sf == nil {
			continue
		}
		for _, r := range refsIn(sf, m.def) {
			if _, taken := roots[r.key()]; !taken {
				roots[r.key()] = m
			}
		}
	}
	return roots
}

// expansion is what a spec block prints and names: the definitions it
// expands, grouped by package in order of first reference, and the ones it
// links or names instead.
type expansion struct {
	order    []string             // package import paths, in order of first expansion
	byPkg    map[string][]*defSrc // expanded definitions per package
	aliasOf  map[string]string    // package -> alias at its first reference
	linked   []Linked
	external []External
}

// expand walks the definitions a member's spec field references,
// transitively. Another member's spec root is linked and anything outside
// the module or vendored is named, not expanded.
func expand(mod *module, m *member, sf *ast.Field, roots map[string]*member) *expansion {
	x := &expansion{byPkg: map[string][]*defSrc{}, aliasOf: map[string]string{}}
	own := map[string]bool{}
	queue := refsIn(sf, m.def)
	for _, r := range queue {
		own[r.key()] = true
	}
	seen := map[string]bool{}
	for len(queue) > 0 {
		r := queue[0]
		queue = queue[1:]
		if seen[r.key()] {
			continue
		}
		seen[r.key()] = true
		if owner, ok := roots[r.key()]; ok && owner != m && !own[r.key()] {
			x.linked = append(x.linked, Linked{Definition: r.display(), Page: owner.page})
			continue
		}
		d, ok := mod.src.lookup(r.pkg, r.name)
		vendored := strings.Contains(stripMajor(r.pkg), vendoredPrefix)
		if !ok || vendored {
			x.external = append(x.external, External{Definition: r.display(), Package: r.pkg, Vendored: vendored})
			continue
		}
		p := d.pkg.importPath
		if _, ok := x.byPkg[p]; !ok {
			x.order = append(x.order, p)
			x.aliasOf[p] = r.alias
		}
		x.byPkg[p] = append(x.byPkg[p], d)
		queue = append(queue, refsIn(d.field.Value, d)...)
	}
	return x
}

// specText prints a member's spec field, then every definition it expands,
// the member's own package first, then each other package in order of
// first reference, with its comments cleaned.
func specText(mod *module, m *member, roots map[string]*member) (code string, linked []Linked, external []External, err error) {
	sf := fieldIn(m.def.field.Value, "spec")
	if sf == nil {
		return "", nil, nil, fmt.Errorf("no spec field in %s", m.def.name)
	}
	x := expand(mod, m, sf, roots)
	var b strings.Builder
	head, err := formatNode(sf)
	if err != nil {
		return "", nil, nil, err
	}
	b.WriteString(head)
	ownPkg := m.def.pkg.importPath
	var pkgs []string
	if _, ok := x.byPkg[ownPkg]; ok {
		pkgs = append(pkgs, ownPkg)
	}
	for _, p := range x.order {
		if p != ownPkg {
			pkgs = append(pkgs, p)
		}
	}
	for _, p := range pkgs {
		if p != ownPkg {
			fmt.Fprintf(&b, "\n\n// Defined in %s, imported as %s.", p, x.aliasOf[p])
		}
		for _, d := range x.byPkg[p] {
			s, err := formatNode(d.field)
			if err != nil {
				return "", nil, nil, err
			}
			b.WriteString("\n\n")
			b.WriteString(s)
		}
	}
	return doctext.CleanCode(b.String()), x.linked, x.external, nil
}

// formatNode prints one declaration with cue/format.
func formatNode(n ast.Node) (string, error) {
	out, err := format.Node(n)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
