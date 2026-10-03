package serve

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-platform-model/docs-kit/internal/dialect"
	"github.com/open-platform-model/docs-kit/internal/render"
)

// site is the skeleton Hugo site: a base template, list and single pages
// and the alert render hook. hugo.json and the figure shortcodes are
// written by writeSite.
//
//go:embed all:site
var site embed.FS

// mount is one bundle's content/ tree mounted into the skeleton's content.
type mount struct {
	Source string `json:"source"` // absolute path of the bundle's content/
	Target string `json:"target"` // "content/docs"
}

// mountTarget is where the skeleton mounts a bundle's content/, so its
// pages publish at the URLs the site gives them: a tab at its root's edge
// segment (/catalogs/opm/edge/), a docs bundle at /docs/, any other
// placement at its root.
func mountTarget(kind, root string) string {
	root = strings.Trim(root, "/")
	switch kind {
	case render.KindTab:
		return "content/" + root + "/edge"
	case render.KindDocs:
		return "content/docs"
	}
	if root == "" {
		return "content"
	}
	return "content/" + root
}

// urlPath is the URL path a mount target serves: "content/docs" is "/docs/".
func urlPath(target string) string {
	p := strings.TrimPrefix(strings.TrimPrefix(target, "content"), "/")
	if p == "" {
		return "/"
	}
	return "/" + p + "/"
}

// hugoConfig is the skeleton's hugo.json: no theme, raw HTML off as on the
// site, no taxonomies, feeds or sitemap.
type hugoConfig struct {
	BaseURL      string   `json:"baseURL"`
	Title        string   `json:"title"`
	DisableKinds []string `json:"disableKinds"`
	Markup       struct {
		Goldmark struct {
			Renderer struct {
				Unsafe bool `json:"unsafe"`
			} `json:"renderer"`
		} `json:"goldmark"`
	} `json:"markup"`
	Module struct {
		Mounts []mount `json:"mounts"`
	} `json:"module"`
}

// writeSite writes the skeleton into dir with the bundles' mounts: the
// embedded files, hugo.json, and one placeholder shortcode per figure the
// dialect allows.
func writeSite(dir string, mounts []mount) error {
	err := fs.WalkDir(site, "site", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(p, "site")))
		if d.IsDir() {
			return os.MkdirAll(dst, 0o750)
		}
		b, err := site.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o600)
	})
	if err != nil {
		return err
	}
	var c hugoConfig
	c.BaseURL = "/"
	c.Title = "opm-docs preview"
	c.DisableKinds = []string{"taxonomy", "term", "rss", "sitemap", "robotstxt"}
	c.Module.Mounts = append([]mount{{Source: "content", Target: "content"}}, mounts...)
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "hugo.json"), append(b, '\n'), 0o600); err != nil {
		return err
	}
	sc := filepath.Join(dir, "layouts", "_shortcodes", "opm")
	if err := os.MkdirAll(sc, 0o750); err != nil {
		return err
	}
	for _, f := range dialect.Figures() {
		body := fmt.Sprintf("<div class=\"figure\"><p>Figure <code>%s</code>: the site draws it here.</p></div>\n", f)
		if err := os.WriteFile(filepath.Join(sc, f+".html"), []byte(body), 0o600); err != nil {
			return err
		}
	}
	return nil
}
