package markdown

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/open-platform-model/docs-kit/internal/gitsrc"
	"github.com/open-platform-model/docs-kit/internal/gittest"
)

const landing = `---
title: "The opm catalog"
description: "The contract."
---

See [traits](/catalogs/opm/4/traits/) and [backup](/catalogs/opm/4/traits/backup/#spec),
[another major](/catalogs/opm/5/traits/), [another catalog](/catalogs/acme/1/),
and [the docs](/docs/concepts/catalogs/).

[ref]: /catalogs/opm/4/resources/
`

func repo(t *testing.T) (*gittest.Repo, Dates) {
	r := gittest.New(t, "")
	r.Write(map[string]string{
		"docs/catalogs/opm/_index.md":      landing,
		"docs/catalogs/opm/guide/howto.md": "---\ntitle: \"How\"\ndescription: \"How.\"\ntype: how-to\n---\n\nText.\n",
	})
	r.Commit("pages")
	g := gitsrc.Repo{Dir: r.Dir}
	head, _ := g.Commit(context.Background(), "HEAD")
	return r, func(ctx context.Context, p string) string { return g.LastMod(ctx, head, p) }
}

func TestCopyRelease(t *testing.T) {
	r, dates := repo(t)
	pages, err := Copy(context.Background(), Options{Root: r.Dir, Dir: "docs/catalogs/opm", Catalog: "/catalogs/opm/", Segment: "4.4", Major: "4", Dates: dates})
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 || pages[0].Path != "_index.md" || pages[1].Path != "guide/howto.md" {
		t.Fatalf("pages %+v", pages)
	}
	body := pages[0].Body
	for _, want := range []string{
		"[traits](/catalogs/opm/4.4/traits/)", "[backup](/catalogs/opm/4.4/traits/backup/#spec)",
		"[another major](/catalogs/opm/5/traits/)", "[another catalog](/catalogs/acme/1/)",
		"[the docs](/docs/concepts/catalogs/)", "[ref]: /catalogs/opm/4.4/resources/",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("landing lacks %q:\n%s", want, body)
		}
	}
	if pages[0].Source != "docs/catalogs/opm/_index.md" || pages[0].Lastmod != gittest.Date || !pages[0].IsLanding() {
		t.Errorf("landing %+v", pages[0])
	}
}

func TestCopyEdge(t *testing.T) {
	r, _ := repo(t)
	pages, err := Copy(context.Background(), Options{Root: r.Dir, Dir: "docs/catalogs/opm", Catalog: "/catalogs/opm/", Segment: "edge"})
	if err != nil {
		t.Fatal(err)
	}
	body := pages[0].Body
	for _, want := range []string{"[traits](/catalogs/opm/edge/traits/)", "[another major](/catalogs/opm/edge/traits/)", "[another catalog](/catalogs/acme/1/)"} {
		if !strings.Contains(body, want) {
			t.Errorf("edge landing lacks %q", want)
		}
	}
	if pages[0].Lastmod != "" {
		t.Errorf("lastmod without a dates source: %q", pages[0].Lastmod)
	}
}

func TestMissingDir(t *testing.T) {
	r, _ := repo(t)
	pages, err := Copy(context.Background(), Options{Root: r.Dir, Dir: "docs/nope", Optional: true})
	if err != nil || pages != nil {
		t.Fatalf("optional missing dir: %v %v", pages, err)
	}
	_, err = Copy(context.Background(), Options{Root: r.Dir, Dir: "docs/nope"})
	var missing *MissingDirError
	if !errors.As(err, &missing) || missing.Dir != "docs/nope" {
		t.Fatalf("err = %v", err)
	}
}
