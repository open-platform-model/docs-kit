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

// TestProbesWithHugo renders each probe of the docs-kit#36 reviews with
// the site's Hugo, as a page of its own: whatever Hugo renders as live
// markup, the check refuses; whatever the check accepts, Hugo renders
// inert or refuses to build.
func TestProbesWithHugo(t *testing.T) {
	bin := hugoBinary(t)
	for name, body := range probes(t) {
		t.Run(name, func(t *testing.T) {
			refused := len(Check(Page{Path: name, Body: body}, Authored)) > 0
			html, built := hugoRender(t, bin, body)
			switch {
			case !built:
				t.Logf("hugo refuses to build it; refused by the check: %v", refused)
			case reActive.Match(html) && !refused:
				t.Fatalf("hugo renders live markup and the check accepts it:\n%s", html)
			case reActive.Match(html):
				t.Log("hugo renders live markup; the check refuses it")
			default:
				t.Logf("hugo renders it inert; refused by the check: %v", refused)
			}
		})
	}
}

// hugoRender builds one page in a fresh site and returns its HTML, or
// false when Hugo refuses to build it.
func hugoRender(t *testing.T, bin string, body []byte) ([]byte, bool) {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{"content/p.md": "---\ntitle: \"p\"\n---\n\n" + string(body)}
	for k, v := range hugoSite {
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
