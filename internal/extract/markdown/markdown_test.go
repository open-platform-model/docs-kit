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

` + "```markdown\nAn example link [traits](/catalogs/opm/4/traits/) stays as written.\n```\n"

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
		"An example link [traits](/catalogs/opm/4/traits/) stays as written.",
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

func TestIncludeExclude(t *testing.T) {
	r := gittest.New(t, "")
	page := "---\ntitle: \"x\"\ndescription: \"x\"\ntype: reference\n---\n"
	r.Write(map[string]string{
		"docs/site/_index.md":                       page,
		"docs/site/reference/operator-resources.md": page,
		"docs/site/reference/cli/opm.md":            page,
		"docs/site/reference/cli/opm-module.md":     page,
		"docs/site/guide/start.md":                  page,
	})
	r.Commit("pages")
	paths := func(o Options) ([]string, error) {
		o.Root, o.Dir = r.Dir, "docs/site"
		pages, err := Copy(context.Background(), o)
		out := make([]string, 0, len(pages))
		for _, p := range pages {
			out = append(out, p.Path)
		}
		return out, err
	}
	for _, c := range []struct {
		name string
		o    Options
		want string
		err  string
	}{
		{name: "one page", o: Options{Include: []string{"reference/operator-resources.md"}}, want: "reference/operator-resources.md"},
		{name: "glob", o: Options{Include: []string{"reference/cli/opm-*.md"}}, want: "reference/cli/opm-module.md"},
		{name: "directory exclude", o: Options{Exclude: []string{"reference/cli/"}},
			want: "_index.md guide/start.md reference/operator-resources.md"},
		{name: "exclude wins over include", o: Options{Include: []string{"reference/"}, Exclude: []string{"reference/cli/opm.md"}},
			want: "reference/cli/opm-module.md reference/operator-resources.md"},
		{name: "no match", o: Options{Exclude: []string{"reference/defintions/"}}, err: `exclude "reference/defintions/" matches no file`},
		{name: "no match in a backfill", o: Options{Exclude: []string{"reference/defintions/"}, Optional: true},
			want: "_index.md guide/start.md reference/cli/opm-module.md reference/cli/opm.md reference/operator-resources.md"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := paths(c.o)
			if c.err != "" {
				if err == nil || !strings.Contains(err.Error(), c.err) || !strings.Contains(err.Error(), "docs/site") {
					t.Fatalf("err = %v, want %q", err, c.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(got, " ") != c.want {
				t.Fatalf("copied %v, want %s", got, c.want)
			}
		})
	}
}

func TestDocsBundleCopiesAsWritten(t *testing.T) {
	r, _ := repo(t)
	pages, err := Copy(context.Background(), Options{Root: r.Dir, Dir: "docs/catalogs/opm"})
	if err != nil {
		t.Fatal(err)
	}
	if pages[0].Body != landing {
		t.Fatalf("a page without a catalog root was rewritten:\n%s", pages[0].Body)
	}
}
