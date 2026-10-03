package mdsafe

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// hugoSite is a site configured as opmodel.dev's markup is (raw HTML on,
// Hugo's goldmark defaults otherwise), with a layout that prints a page's
// content alone.
var hugoSite = map[string]string{
	"hugo.toml": "baseURL = \"https://example.org/\"\n" +
		"disableKinds = [\"taxonomy\", \"term\", \"RSS\", \"sitemap\", \"robotsTXT\", \"home\", \"section\", \"404\"]\n" +
		"[markup.goldmark.renderer]\n  unsafe = true\n",
	"layouts/_default/single.html": "{{ .Content }}",
}

// hugoBinary is the hugo on PATH, at the version the check is pinned to.
// The test skips without one, unless OPM_DOCS_REQUIRE_HUGO is set (CI's
// job that installs the site's pinned Hugo).
func hugoBinary(t *testing.T) string {
	t.Helper()
	required := os.Getenv("OPM_DOCS_REQUIRE_HUGO") != ""
	bin, err := exec.LookPath("hugo")
	if err != nil {
		if required {
			t.Fatal("OPM_DOCS_REQUIRE_HUGO is set and there is no hugo on PATH")
		}
		t.Skip("no hugo on PATH")
	}
	out, err := exec.CommandContext(context.Background(), bin, "version").Output()
	if err != nil || !strings.Contains(string(out), "v"+HugoVersion+"-") {
		if required {
			t.Fatalf("hugo version %q, want %s: %v", out, HugoVersion, err)
		}
		t.Skipf("hugo on PATH is not %s: %s", HugoVersion, out)
	}
	return bin
}

// hugoBuildFails names the probes whose page Hugo refuses to build. A
// probe that fails to build and is not named here fails the test: a build
// failure says nothing about what the check accepts.
var hugoBuildFails = map[string]bool{}

// TestProbesWithHugo renders each probe with the site's Hugo, as a page of
// its own: whatever Hugo renders as live markup, the check refuses in both
// modes.
func TestProbesWithHugo(t *testing.T) {
	bin := hugoBinary(t)
	for name, body := range probes(t) {
		t.Run(name, func(t *testing.T) {
			h, built := hugoRender(t, bin, hugoSite, body)
			if !built {
				if !hugoBuildFails[name] {
					t.Fatal("hugo refuses to build it, and hugoBuildFails does not name it")
				}
				return
			}
			judge(t, name, body, h, nil)
		})
	}
}

// judge fails when html is live and a mode accepts the probe.
func judge(t *testing.T, name string, body, h []byte, allow map[string]bool) {
	t.Helper()
	a := active(h, allow)
	for mode, m := range modes {
		refused := len(Check(Page{Path: name, Body: body}, mode)) > 0
		switch {
		case a != "" && !refused:
			t.Fatalf("%s: hugo renders %s and the check accepts it:\n%s", m, a, h)
		case a != "":
			t.Logf("%s: hugo renders %s; the check refuses it", m, a)
		default:
			t.Logf("%s: hugo renders it inert; refused by the check: %v", m, refused)
		}
	}
}

// siteAllow is what the pinned Hextra theme's and opmodel.dev's _markup
// hooks write beside goldmark's elements (heading anchors, link icons, code
// block chrome).
var siteAllow = map[string]bool{"span": true, "svg": true, "path": true, "button": true, "figure": true, "figcaption": true}

// TestProbesWithSite renders each probe again through the pinned Hextra
// theme and opmodel.dev's _markup hooks, with the site's minifier, when
// OPM_DOCS_SITE_DIR names an opmodel.dev site/ directory. The site's link
// hooks fail the build on a link they cannot resolve; a probe whose build
// fails must be refused by the check in both modes.
func TestProbesWithSite(t *testing.T) {
	dir := os.Getenv("OPM_DOCS_SITE_DIR")
	if dir == "" {
		t.Skip("OPM_DOCS_SITE_DIR names no opmodel.dev site/ directory")
	}
	bin := hugoBinary(t)
	themes, err := filepath.Abs(filepath.Join(dir, "themes"))
	if err != nil {
		t.Fatal(err)
	}
	site := map[string]string{
		"hugo.toml": "baseURL = \"https://example.org/\"\ntheme = \"hextra\"\nthemesDir = \"" + themes + "\"\n" +
			"disableKinds = [\"taxonomy\", \"term\", \"RSS\", \"sitemap\", \"robotsTXT\", \"home\", \"section\", \"404\"]\n" +
			"[minify]\n  disableSVG = true\n  minifyOutput = true\n" +
			"[markup.goldmark.renderer]\n  unsafe = true\n",
		"layouts/_default/single.html": "{{ .Content }}",
	}
	hooks, err := filepath.Glob(filepath.Join(dir, "layouts", "_markup", "*.html"))
	if err != nil || len(hooks) == 0 {
		t.Fatalf("no _markup hooks under %s: %v", dir, err)
	}
	for _, f := range hooks {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		site["layouts/_markup/"+filepath.Base(f)] = string(b)
	}
	for name, body := range probes(t) {
		t.Run(name, func(t *testing.T) {
			h, built := hugoRender(t, bin, site, body)
			if !built {
				for mode, m := range modes {
					if len(Check(Page{Path: name, Body: body}, mode)) == 0 {
						t.Fatalf("%s: the site's hooks refuse to build it and the check accepts it", m)
					}
				}
				return
			}
			judge(t, name, body, h, siteAllow)
		})
	}
}

// hugoRender builds one page in a fresh site and returns its HTML, or
// false when Hugo refuses to build it.
func hugoRender(t *testing.T, bin string, site map[string]string, body []byte) ([]byte, bool) {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{"content/p.md": "---\ntitle: \"p\"\n---\n\n" + string(body)}
	for k, v := range site {
		files[k] = v
	}
	for p, b := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(b), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.CommandContext(context.Background(), bin, "--quiet", "--source", dir, "--destination", filepath.Join(dir, "public"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Logf("hugo: %v: %s", err, out)
		return nil, false
	}
	html, err := os.ReadFile(filepath.Join(dir, "public", "p", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	return html, true
}
