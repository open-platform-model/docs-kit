package config

import (
	"strings"
	"testing"
)

// enhancementsConfig is the enhancements repository's docs-kit.cue.
const enhancementsConfig = `bundles: enhancements: {
	placement: {kind: "section", root: "/enhancements/"}
	sources: [{kind: "enhancements", description: "OPM's design record: every proposal, its decisions and its status."}]
}
`

func TestLoadEnhancementsSection(t *testing.T) {
	c, err := Load(write(t, "docs-kit.cue", enhancementsConfig))
	if err != nil {
		t.Fatal(err)
	}
	b := c.Bundles["enhancements"]
	if b.Placement.Kind != PlacementSection || b.Placement.Root != "/enhancements/" || b.Version.Prefix != "" || b.Sources[0].Kind != KindEnhancements {
		t.Fatalf("decoded %+v", b)
	}
	var opts struct{ Dir, Title, Description string }
	if err := b.Sources[0].Value.Decode(&opts); err != nil || opts.Dir != "." || opts.Title != "Enhancements" {
		t.Fatalf("defaults %+v: %v", opts, err)
	}
	for _, x := range []struct{ name, body, msg string }{
		{"an enhancements source in a docs bundle", strings.Replace(enhancementsConfig, `{kind: "section", root: "/enhancements/"}`, `{kind: "docs", root: "/docs/"}
	version: {from: "tag", prefix: "v"}`, 1), `give the bundle placement {kind: "section"`},
		{"a markdown source in a section bundle", strings.Replace(enhancementsConfig, `sources: [`, `sources: [{kind: "markdown", dir: "docs"}, `, 1), "holds only an enhancements source, and this one is markdown"},
		{"another section root", strings.Replace(enhancementsConfig, `root: "/enhancements/"`, `root: "/rfcs/"`, 1), "placement"},
		{"no description", strings.Replace(enhancementsConfig, `, description: "OPM's design record: every proposal, its decisions and its status."`, "", 1), "description"},
		{"an absolute dir", strings.Replace(enhancementsConfig, `kind: "enhancements",`, `kind: "enhancements", dir: "/etc",`, 1), "dir"},
		{"a dir that climbs", strings.Replace(enhancementsConfig, `kind: "enhancements",`, `kind: "enhancements", dir: "../x",`, 1), "dir"},
		{"citations", strings.Replace(enhancementsConfig, `kind: "enhancements",`, `kind: "enhancements", citations: "strip",`, 1), "citations"},
	} {
		t.Run(x.name, func(t *testing.T) {
			_, err := Load(write(t, "docs-kit.cue", x.body))
			if err == nil || !strings.Contains(err.Error(), x.msg) {
				t.Fatalf("err = %v, want it to name %q", err, x.msg)
			}
		})
	}
}
