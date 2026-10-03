package tags

import (
	"slices"
	"testing"
)

func build(t *testing.T, v string, r int) Build {
	t.Helper()
	b, err := NewBuild(v, r, "sha256:"+v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseVersion(t *testing.T) {
	for _, ok := range []string{"4.4.5", "0.1.0", "1.0.0-beta.5", "4.5.0-rc.1", "1.0.0-alpha-1.x"} {
		if _, err := ParseVersion(ok); err != nil {
			t.Errorf("%s: %v", ok, err)
		}
	}
	for _, bad := range []string{"v4.4.5", "4.4", "4.4.5+build", "04.4.5", "4.4.5-01", "edge", ""} {
		if _, err := ParseVersion(bad); err == nil {
			t.Errorf("%s parsed", bad)
		}
	}
}

func TestOrder(t *testing.T) {
	// Every pair in ascending order.
	asc := [][2]any{
		{"1.0.0-alpha", 0}, {"1.0.0-alpha.1", 0}, {"1.0.0-alpha.beta", 0}, {"1.0.0-beta", 0},
		{"1.0.0-beta.2", 0}, {"1.0.0-beta.5", 0}, {"1.0.0-beta.5", 1}, {"1.0.0-beta.11", 0},
		{"1.0.0-rc.1", 0}, {"1.0.0", 0}, {"4.4.5", 0}, {"4.4.5", 1}, {"4.4.6", 0},
		{"4.5.0-rc.1", 0}, {"4.5.0", 0}, {"4.10.0", 0},
	}
	bs := make([]Build, 0, len(asc))
	for _, a := range asc {
		bs = append(bs, build(t, a[0].(string), a[1].(int)))
	}
	for i := range bs {
		for j := range bs {
			want := cmpInt(i, j)
			if got := CompareBuilds(bs[i], bs[j]); got != want {
				t.Errorf("CompareBuilds(%s, %s) = %d, want %d", bs[i], bs[j], got, want)
			}
		}
	}
}

func TestNames(t *testing.T) {
	cases := []struct {
		version       string
		revision      int
		full, segment string
		moving        []string
	}{
		{"4.4.5", 0, "4.4.5.0", "4.4", []string{"4.4.5", "4.4", "4"}},
		{"4.4.5", 1, "4.4.5.1", "4.4", []string{"4.4.5", "4.4", "4"}},
		{"1.0.0-beta.2", 0, "1.0.0-beta.2.0", "1.0", []string{"1.0.0-beta.2", "1.0", "1"}},
		{"1.0.0-beta.5", 0, "1.0.0-beta.5.0", "1.0", []string{"1.0.0-beta.5", "1.0", "1"}},
		{"edge", 0, "", "edge", []string{"edge"}},
	}
	for _, c := range cases {
		b := build(t, c.version, c.revision)
		var moving []string
		for _, l := range Lines(b) {
			moving = append(moving, l.Tag)
		}
		if b.FullTag() != c.full || b.Segment() != c.segment || !slices.Equal(moving, c.moving) {
			t.Errorf("%s.%d: full %q segment %q moving %v", c.version, c.revision, b.FullTag(), b.Segment(), moving)
		}
	}
	if _, err := NewBuild("edge", 1, ""); err == nil {
		t.Error("edge revision 1 accepted")
	}
}

func TestPromotion(t *testing.T) {
	cases := []struct {
		name     string
		existing [][2]any
		d        [2]any
		want     []string
	}{
		{"first release", nil, [2]any{"4.4.5", 0}, []string{"4.4.5", "4.4", "4"}},
		{"new patch", [][2]any{{"4.4.5", 0}}, [2]any{"4.4.6", 0}, []string{"4.4.6", "4.4", "4"}},
		{"revision of an older patch", [][2]any{{"4.4.5", 0}, {"4.4.5", 1}, {"4.4.6", 0}}, [2]any{"4.4.5", 2}, []string{"4.4.5"}},
		{"concurrent: newer minor already holds the major", [][2]any{{"4.5.0", 0}}, [2]any{"4.4.6", 0}, []string{"4.4.6", "4.4"}},
		{"revision promoted", [][2]any{{"4.4.5", 0}}, [2]any{"4.4.5", 1}, []string{"4.4.5", "4.4", "4"}},
		{"prerelease of the next minor", [][2]any{{"4.4.6", 0}}, [2]any{"4.5.0-rc.1", 0}, []string{"4.5.0-rc.1", "4.5", "4"}},
		{"re-run of the newest", [][2]any{{"4.4.5", 0}}, [2]any{"4.4.5", 0}, []string{"4.4.5", "4.4", "4"}},
		{"release after its prerelease", [][2]any{{"1.0.0-beta.5", 0}}, [2]any{"1.0.0", 0}, []string{"1.0.0", "1.0", "1"}},
		{"prerelease after its release", [][2]any{{"1.0.0", 0}}, [2]any{"1.0.0-beta.5", 0}, []string{"1.0.0-beta.5"}},
		{"edge", [][2]any{{"4.4.5", 0}}, [2]any{"edge", 0}, []string{"edge"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var bs []Build
			for _, e := range c.existing {
				bs = append(bs, build(t, e[0].(string), e[1].(int)))
			}
			got := Promotion(build(t, c.d[0].(string), c.d[1].(int)), bs)
			if !slices.Equal(got, c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestNextRevision(t *testing.T) {
	v, _ := ParseVersion("4.4.5")
	bs := []Build{build(t, "4.4.5", 0), build(t, "4.4.5", 1), build(t, "4.4.6", 0), build(t, "edge", 0)}
	if r, err := NextRevision(v, bs); err != nil || r != 2 {
		t.Fatalf("got %d %v", r, err)
	}
	w, _ := ParseVersion("4.4.7")
	if _, err := NextRevision(w, bs); err == nil {
		t.Fatal("revision of an unpublished release accepted")
	}
	// A revision without its revision 0 is no base for another.
	x, _ := ParseVersion("4.4.8")
	if _, err := NextRevision(x, append(bs, build(t, "4.4.8", 1))); err == nil {
		t.Fatal("revision of a release without revision 0 accepted")
	}
}

func TestTagForms(t *testing.T) {
	for tag, want := range map[string]bool{"4.4": true, "10.0": true, "4": false, "4.4.5": false, "edge": false, "04.4": false} {
		if IsMinorTag(tag) != want {
			t.Errorf("IsMinorTag(%q) != %v", tag, want)
		}
	}
	for tag, want := range map[string][2]any{"4.4.5.0": {"4.4.5", 0}, "1.0.0-beta.2.0": {"1.0.0-beta.2", 0}, "4.4.5.12": {"4.4.5", 12}} {
		v, r, ok := SplitFullTag(tag)
		if !ok || v != want[0] || r != want[1] {
			t.Errorf("SplitFullTag(%q) = %q %d %v", tag, v, r, ok)
		}
	}
	for _, tag := range []string{"4.4.5", "4.4", "4", "edge", "sha256-abc"} {
		if _, _, ok := SplitFullTag(tag); ok {
			t.Errorf("SplitFullTag(%q) accepted", tag)
		}
	}
	if !IsSignatureTag("sha256-3891fc") || IsSignatureTag("4.4") {
		t.Error("IsSignatureTag")
	}
	segs := []string{"edge", "4.10", "4.4", "10.0", "4.5"}
	slices.SortFunc(segs, CompareMinor)
	if !slices.Equal(segs, []string{"4.4", "4.5", "4.10", "10.0", "edge"}) {
		t.Errorf("sorted %v", segs)
	}
}
