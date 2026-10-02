package bundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var created = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

func TestReadFixture(t *testing.T) {
	m, err := Read("testdata/tree")
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckTree("testdata/tree", m); err != nil {
		t.Fatal(err)
	}
	if m.Segment() != "4.4" || m.Annotations()[AnnSource] != "https://github.com/open-platform-model/catalog_opm" {
		t.Fatalf("segment %s annotations %v", m.Segment(), m.Annotations())
	}
	enc, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile("testdata/tree/manifest.json")
	if !bytes.Equal(enc, raw) {
		t.Fatalf("encoding changed the key order or layout:\n%s", enc)
	}
}

func TestManifestSchema(t *testing.T) {
	raw, _ := os.ReadFile("testdata/tree/manifest.json")
	cases := []struct {
		name, from, to, want string
	}{
		{"edge with a revision", `"version": "4.4.5",
  "revision": 0`, `"version": "edge",
  "revision": 1`, "revision"},
		{"unknown field", `"tool": "0.1.0"`, `"tool": "0.1.0", "extra": 1`, "extra"},
		{"short commit", `"0123456789abcdef0123456789abcdef01234567"`, `"0123456"`, "commit"},
		{"docs placement with a tab root", `"kind": "tab"`, `"kind": "docs"`, "root"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bad := strings.Replace(string(raw), c.from, c.to, 1)
			if bad == string(raw) {
				t.Fatal("replacement did not apply")
			}
			_, err := Parse("manifest.json", []byte(bad))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want one naming %q", err, c.want)
			}
		})
	}
}

func TestUnlistedPage(t *testing.T) {
	dir := copyTree(t)
	if err := os.WriteFile(filepath.Join(dir, "content", "traits", "extra.md"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	err = CheckTree(dir, m)
	if err == nil || !strings.Contains(err.Error(), "content/traits/extra.md is not listed") {
		t.Fatalf("err = %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "content", "traits", "backup.md")); err != nil {
		t.Fatal(err)
	}
	if err := CheckTree(dir, m); err == nil || !strings.Contains(err.Error(), "lists content/traits/backup.md, which does not exist") {
		t.Fatalf("err = %v", err)
	}
}

func copyTree(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir("testdata/tree", func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel("testdata/tree", p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), b, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

// pinnedLayer is the digest of the fixture tree's layer. It changes only
// when the packing rules, the fixture or Go's compress/flate change.
const pinnedLayer = "sha256:e4fe7cd1ee49ff140e80c9d4c97a0f8f2ad59dd6a77531d9e1f11d7ff3f356f5"

func TestPackDeterministic(t *testing.T) {
	a, err := Pack("testdata/tree", created)
	if err != nil {
		t.Fatal(err)
	}
	// A copy with other file modes and times packs to the same bytes.
	dir := copyTree(t)
	_ = os.Chmod(filepath.Join(dir, "content", "_index.md"), 0o600)
	_ = os.Chtimes(filepath.Join(dir, "data", "catalog.json"), time.Now(), time.Now())
	b, err := Pack(dir, created)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("two packs of one tree differ")
	}
	sum := sha256.Sum256(a)
	if got := "sha256:" + hex.EncodeToString(sum[:]); got != pinnedLayer {
		t.Fatalf("layer digest %s, pinned %s", got, pinnedLayer)
	}
	tr := tar.NewReader(mustGunzip(t, a))
	var names []string
	for {
		h, err := tr.Next()
		if err != nil {
			break
		}
		if h.Uid != 0 || h.Gid != 0 || h.Uname != "" || h.Gname != "" || !h.ModTime.Equal(created) || len(h.PAXRecords) > 0 {
			t.Errorf("%s: header %+v", h.Name, h)
		}
		names = append(names, h.Name)
	}
	want := "content/ content/_index.md content/traits/ content/traits/backup.md data/ data/catalog.json manifest.json"
	if strings.Join(names, " ") != want {
		t.Fatalf("entries %v", names)
	}
}

func mustGunzip(t *testing.T, b []byte) *bytes.Reader {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if zr.Name != "" || !zr.ModTime.IsZero() {
		t.Fatalf("gzip header name %q time %v", zr.Name, zr.ModTime)
	}
	var out bytes.Buffer
	if _, err := out.ReadFrom(zr); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(out.Bytes())
}

func TestUnpackRoundTrip(t *testing.T) {
	layer, err := Pack("testdata/tree", created)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "4.4")
	m, err := Unpack(bytes.NewReader(layer), dest, DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	if m.Version != "4.4.5" {
		t.Fatalf("manifest %+v", m)
	}
	if _, err := os.Stat(filepath.Join(dest, "content", "traits", "backup.md")); err != nil {
		t.Fatal(err)
	}
}

type tarEntry struct {
	name     string
	typeflag byte
	body     string
	link     string
}

func layerOf(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		h := &tar.Header{Name: e.name, Typeflag: e.typeflag, Mode: 0o644, Size: int64(len(e.body)), Linkname: e.link}
		if e.typeflag != tar.TypeReg {
			h.Size = 0
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Size > 0 {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

func TestUnpackRefuses(t *testing.T) {
	manifest, _ := os.ReadFile("testdata/tree/manifest.json")
	good := []tarEntry{
		{name: "manifest.json", typeflag: tar.TypeReg, body: string(manifest)},
		{name: "content/_index.md", typeflag: tar.TypeReg, body: "x"},
		{name: "content/traits/backup.md", typeflag: tar.TypeReg, body: "x"},
		{name: "data/catalog.json", typeflag: tar.TypeReg, body: "{}"},
	}
	with := func(extra ...tarEntry) []tarEntry { return append(append([]tarEntry{}, good...), extra...) }
	cases := []struct {
		name    string
		entries []tarEntry
		lim     Limits
		want    string
	}{
		{"traversal", with(tarEntry{name: "content/../../etc/passwd", typeflag: tar.TypeReg, body: "x"}), DefaultLimits, `content/../../etc/passwd: path element ".."`},
		{"absolute", with(tarEntry{name: "/etc/passwd", typeflag: tar.TypeReg, body: "x"}), DefaultLimits, "absolute path"},
		{"symlink", with(tarEntry{name: "content/link.md", typeflag: tar.TypeSymlink, link: "/etc/passwd"}), DefaultLimits, "symlink"},
		{"hard link", with(tarEntry{name: "content/link.md", typeflag: tar.TypeLink, link: "manifest.json"}), DefaultLimits, "hard link"},
		{"device", with(tarEntry{name: "content/dev", typeflag: tar.TypeChar}), DefaultLimits, "entry type"},
		{"fifo", with(tarEntry{name: "content/fifo", typeflag: tar.TypeFifo}), DefaultLimits, "entry type"},
		{"duplicate", with(tarEntry{name: "content/_index.md", typeflag: tar.TypeReg, body: "y"}), DefaultLimits, "duplicate"},
		{"extra top level", with(tarEntry{name: "extra.txt", typeflag: tar.TypeReg, body: "x"}), DefaultLimits, "not allowed at the top level"},
		{"too many entries", good, Limits{Entries: 3, Bytes: 1 << 20}, "more than 3 entries"},
		{"too many bytes", good, Limits{Entries: 100, Bytes: 10}, "more than 10 bytes"},
		{"unlisted file", with(tarEntry{name: "content/extra.md", typeflag: tar.TypeReg, body: "x"}), DefaultLimits, "content/extra.md is not listed"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parent := t.TempDir()
			dest := filepath.Join(parent, "4.4")
			_, err := Unpack(bytes.NewReader(layerOf(t, c.entries)), dest, c.lim)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want one containing %q", err, c.want)
			}
			if _, err := os.Lstat(dest); !os.IsNotExist(err) {
				t.Fatalf("%s exists after a refused unpack", dest)
			}
			left, _ := os.ReadDir(parent)
			if len(left) != 0 {
				t.Fatalf("unpack left %v behind", left)
			}
		})
	}
}
