package cuecatalog_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/extract/cuecatalog"
	"github.com/open-platform-model/docs-kit/internal/render"
)

// TestCatalogOPMParity renders the opm catalog of a catalog_opm checkout and
// compares every member page with the page catalog_opm's own generator
// committed at that commit, after normalizing the differences the bundle
// format makes on purpose: no marker comments, links in the bundle's own
// segment, the contract link pointing at the landing, and the newest
// apiVersion of a name on the bare page path. Opt-in: it needs the
// checkout and GHCR reads, so it runs only when OPM_DOCS_CATALOG_OPM names
// the checkout.
func TestCatalogOPMParity(t *testing.T) {
	root := os.Getenv("OPM_DOCS_CATALOG_OPM")
	if root == "" {
		t.Skip("OPM_DOCS_CATALOG_OPM is unset")
	}
	m, err := cuecatalog.Extract(cuecatalog.Options{Root: root, Module: "./opm"})
	if err != nil {
		t.Fatal(err)
	}
	segment := strings.Join(strings.SplitN(m.Version, ".", 3)[:2], ".")
	tg := render.Target{Root: "/catalogs/opm/", Segment: segment, Repo: "open-platform-model/catalog_opm", Commit: strings.Repeat("0", 40)}
	pages, err := render.Catalog(m, tg)
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]string{}
	for _, p := range pages {
		body[p.Path] = p.Body
	}
	refgenPage := refgenPages(m)
	compared := 0
	for i := range m.Members {
		mem := &m.Members[i]
		old := ""
		for k, v := range refgenPage {
			if v == mem.Page {
				old = k
			}
		}
		committed, err := os.ReadFile(filepath.Join(root, "docs", "site", "reference", "catalog-members", old+".md"))
		if err != nil {
			t.Errorf("%s: %v", mem.FQN, err)
			continue
		}
		want := normalize(string(committed), refgenPage, tg)
		got := strings.TrimSpace(body[mem.Page+".md"])
		compared++
		if got != want {
			t.Errorf("%s: the page differs from refgen's %s.md\n--- refgen\n%s\n--- opm-docs\n%s", mem.FQN, old, firstDiff(want), firstDiff(got))
		}
	}
	if compared != len(m.Members) || compared == 0 {
		t.Fatalf("compared %d of %d member pages", compared, len(m.Members))
	}
	t.Logf("%d member pages match refgen's", compared)
}

// refgenPages maps refgen's page paths to this renderer's: refgen names
// every apiVersion of a shared name <name>-<apiVersion>.
func refgenPages(m *cuecatalog.Model) map[string]string {
	count := map[string]int{}
	for i := range m.Members {
		count[m.Members[i].Kind+"/"+m.Members[i].Name]++
	}
	out := map[string]string{}
	for i := range m.Members {
		mem := &m.Members[i]
		old := mem.Kind + "s/" + mem.Name
		if count[mem.Kind+"/"+mem.Name] > 1 {
			old += "-" + mem.APIVersion
		}
		out[old] = mem.Page
	}
	return out
}

var reLink = regexp.MustCompile(`/docs/reference/catalog-members/([a-z]+/[a-z0-9-]+)/`)

// normalize turns a refgen page into the form this renderer writes.
func normalize(s string, pages map[string]string, tg render.Target) string {
	var keep []string
	for _, l := range strings.Split(s, "\n") {
		if strings.HasPrefix(l, "<!-- BEGIN GENERATED") || strings.HasPrefix(l, "<!-- END GENERATED") {
			continue
		}
		keep = append(keep, l)
	}
	s = strings.Join(keep, "\n")
	s = strings.ReplaceAll(s, "---\n\n\n## At a glance", "---\n\n## At a glance")
	s = reLink.ReplaceAllStringFunc(s, func(l string) string {
		if p, ok := pages[reLink.FindStringSubmatch(l)[1]]; ok {
			return tg.URL(p)
		}
		return l
	})
	s = strings.ReplaceAll(s, "/docs/reference/catalog-contract/", tg.URL(""))
	return strings.TrimSpace(s)
}

// firstDiff keeps a failure readable: the page's first 40 lines.
func firstDiff(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) > 40 {
		lines = lines[:40]
	}
	return strings.Join(lines, "\n")
}
