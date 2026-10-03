package config

import (
	"strings"
	"testing"
)

const sectionsPull = `sections: enhancements: {repo: "open-platform-model/enhancements", root: "/enhancements/"}
`

func TestLoadPullSections(t *testing.T) {
	p, err := LoadPull(write(t, "bundles.cue", siteConfig+sectionsPull))
	if err != nil {
		t.Fatal(err)
	}
	if s := p.Sections["enhancements"]; s.Repo != "open-platform-model/enhancements" || s.Root != "/enhancements/" || len(p.SectionProjects()) != 1 {
		t.Fatalf("sections %+v", p.Sections)
	}
	for _, x := range []struct{ name, body, msg string }{
		{"a section and a docs project", siteConfig + sectionsPull + `docs: enhancements: {repo: "open-platform-model/enhancements"}
`, "enhancements is both a section and a docs project"},
		{"a section and a tab", siteConfig + sectionsPull + `tabs: enhancements: {repo: "open-platform-model/enhancements", root: "/catalogs/enh/", from: "1.0"}
`, "enhancements is both a section and a tab project"},
		{"two sections on one root", siteConfig + sectionsPull + `sections: rfcs: {repo: "open-platform-model/rfcs", root: "/enhancements/"}
`, "sections enhancements and rfcs both claim the root /enhancements/"},
		{"another root", strings.Replace(sectionsPull, `root: "/enhancements/"`, `root: "/rfcs/"`, 1), "root"},
		{"no repo", `sections: enhancements: {root: "/enhancements/"}
`, "repo"},
	} {
		t.Run(x.name, func(t *testing.T) {
			_, err := LoadPull(write(t, "bundles.cue", x.body))
			if err == nil || !strings.Contains(err.Error(), x.msg) {
				t.Fatalf("err = %v, want it to name %q", err, x.msg)
			}
		})
	}
}
