package pull

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/open-platform-model/docs-kit/internal/bundle"
	"github.com/open-platform-model/docs-kit/internal/placeholder"
)

// fillVersion fills the version placeholders of one staged site version. It
// runs after the overlap check and before anything is swapped in. Each page
// a docs bundle lists is rewritten in place by a plain text replace; no other
// file is touched. A value the site version cannot supply refuses the pull.
func (p *puller) fillVersion(sv string, set []*docsBundle) error {
	values, edge, err := placeholderValues(sv, set)
	if err != nil {
		return err
	}
	for _, b := range set {
		for _, page := range b.m.Pages {
			file := filepath.Join(b.dir, bundle.ContentDir, filepath.FromSlash(page.Path))
			raw, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			if !strings.Contains(string(raw), placeholder.Open) {
				continue
			}
			out, left := placeholder.Fill(string(raw), values)
			if len(left) > 0 {
				return fmt.Errorf("%s: %s: %s: %s", sv, b.what, page.Path, unfilled(left[0], edge))
			}
			if err := os.WriteFile(file, []byte(out), 0o600); err != nil { //nolint:gosec // a listed page of the staged tree: the manifest schema fixes its path, and the file exists, so its mode is kept
				return err
			}
		}
	}
	return nil
}

// placeholderValues builds the table of one site version: a bundle name takes
// the version of the bundle of its docs project, a unit name the entry a
// bundle's manifest records. A name whose bundle is at edge is no value; it
// is named in the second result so the refusal can say why. Two bundles
// recording one unit are refused.
func placeholderValues(sv string, set []*docsBundle) (values, edge map[string]string, err error) {
	values = map[string]string{}
	edge = map[string]string{}
	for _, n := range placeholder.All() {
		switch n.Kind {
		case placeholder.Bundle:
			for _, b := range set {
				if b.project != n.Project {
					continue
				}
				if b.m.Version == "edge" {
					edge[n.Name] = b.what
				} else {
					values[n.Name] = b.m.Version
				}
			}
		case placeholder.Unit:
			var from []string
			for _, b := range set {
				if v, ok := b.m.Units[n.Name]; ok {
					values[n.Name] = v
					from = append(from, b.what)
				}
			}
			if len(from) > 1 {
				sort.Strings(from)
				return nil, nil, fmt.Errorf("%s: %s and %s both record the unit %s; one bundle of a site version records it, so drop it from the config of one", sv, from[0], from[1], n.Name)
			}
		}
	}
	return values, edge, nil
}

// unfilled says why an occurrence was left after the fill.
func unfilled(o placeholder.Occurrence, edge map[string]string) string {
	if o.Verdict != placeholder.Known {
		return o.Message()
	}
	n, _ := placeholder.Lookup(o.Name)
	switch {
	case edge[o.Name] != "":
		return fmt.Sprintf("placeholder %q: the value is edge, not a version (%s was pulled at tag edge); pull a release of it", o.Text, edge[o.Name])
	case n.Kind == placeholder.Unit:
		return fmt.Sprintf("placeholder %q: no bundle of this site version records the unit %s; declare it in units of the bundle's docs-kit.cue, or drop the placeholder", o.Text, o.Name)
	}
	return fmt.Sprintf("placeholder %q: this site version pulls no %s bundle; add %s to the site version in the pull config, or drop the placeholder", o.Text, n.Project, n.Project)
}

// checkLeftover scans every page of every staged tree, tabs, sections and
// site versions, for the placeholder opening. It runs once all fills are
// done and before any swap, so a leftover leaves the previous trees alone.
func (p *puller) checkLeftover() error {
	type tree struct{ label, dir string }
	var trees []tree
	for _, key := range p.order {
		trees = append(trees, tree{key, p.staged[key]})
	}
	for _, sv := range p.versionOrder {
		es, err := os.ReadDir(p.versionStaged[sv])
		if err != nil {
			return err
		}
		for _, e := range es {
			if e.IsDir() {
				trees = append(trees, tree{sv + "/" + e.Name(), filepath.Join(p.versionStaged[sv], e.Name())})
			}
		}
	}
	for _, t := range trees {
		m, err := bundle.Read(t.dir)
		if err != nil {
			return err
		}
		for _, page := range m.Pages {
			raw, err := os.ReadFile(filepath.Join(t.dir, bundle.ContentDir, filepath.FromSlash(page.Path)))
			if err != nil {
				return err
			}
			if occ := placeholder.Scan(string(raw)); len(occ) > 0 {
				return fmt.Errorf("%s: %s: placeholder left after the fill: %q; only the docs bundles of a site version are filled", t.label, page.Path, occ[0].Text)
			}
		}
	}
	return nil
}
