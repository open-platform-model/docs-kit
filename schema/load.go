package schema

import (
	"fmt"
	"io/fs"
	"path"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
)

// virtualDir is where the embedded files appear to the CUE loader. Nothing
// is read from disk under it.
const virtualDir = "/opm-docs-schema"

var (
	once   sync.Once
	ctx    *cue.Context
	pkg    cue.Value
	errPkg error
)

// Package returns the evaluated schema package and the context it lives in.
// Values compiled for validation must use the same context.
func Package() (*cue.Context, cue.Value, error) {
	once.Do(func() {
		ctx = cuecontext.New()
		overlay := map[string]load.Source{}
		names, err := fs.Glob(Files, "*.cue")
		if err != nil {
			errPkg = err
			return
		}
		for _, n := range names {
			b, err := Files.ReadFile(n)
			if err != nil {
				errPkg = err
				return
			}
			overlay[path.Join(virtualDir, n)] = load.FromBytes(b)
		}
		insts := load.Instances([]string{"."}, &load.Config{Dir: virtualDir, Overlay: overlay})
		if len(insts) != 1 || insts[0].Err != nil {
			errPkg = fmt.Errorf("loading the embedded schema: %w", insts[0].Err)
			return
		}
		pkg = ctx.BuildInstance(insts[0])
		errPkg = pkg.Err()
	})
	return ctx, pkg, errPkg
}

// Def returns one definition of the schema package, such as "#Manifest".
func Def(name string) (*cue.Context, cue.Value, error) {
	c, p, err := Package()
	if err != nil {
		return nil, cue.Value{}, err
	}
	d := p.LookupPath(cue.ParsePath(name))
	if !d.Exists() {
		return nil, cue.Value{}, fmt.Errorf("schema has no %s", name)
	}
	return c, d, nil
}

// SourceKinds lists the source kinds #Source admits, in schema order: the
// kind of each branch of its disjunction.
func SourceKinds() ([]string, error) {
	_, d, err := Def("#Source")
	if err != nil {
		return nil, err
	}
	branches := []cue.Value{d}
	if op, args := d.Expr(); op == cue.OrOp {
		branches = args
	}
	kinds := make([]string, 0, len(branches))
	for _, b := range branches {
		k, err := b.LookupPath(cue.ParsePath("kind")).String()
		if err != nil {
			return nil, fmt.Errorf("a #Source branch has no concrete kind: %w", err)
		}
		kinds = append(kinds, k)
	}
	return kinds, nil
}
