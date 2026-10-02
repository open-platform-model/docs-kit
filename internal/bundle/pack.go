package bundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Pack builds the bundle layer from dir: a gzip-compressed tar with
// entries sorted by path, directories before their contents, mode 0644 for
// files and 0755 for directories, uid and gid 0 with no owner names, every
// time equal to created, no extended headers, and a gzip header with no
// name and no time. The same tree and created time give the same bytes.
func Pack(dir string, created time.Time) ([]byte, error) {
	entries, err := packEntries(dir)
	if err != nil {
		return nil, err
	}
	created = created.UTC()
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		if err := writeEntry(tw, e, created); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type packEntry struct {
	name string // slash-separated; a directory ends in "/"
	path string
	dir  bool
}

// packEntries lists the tree in byte order of the entry names, which puts
// "a/" before "a/b" and so every directory before its contents.
func packEntries(dir string) ([]packEntry, error) {
	var entries []packEntry
	for _, top := range []string{ManifestFile, ContentDir, DataDir} {
		root := filepath.Join(dir, top)
		if _, err := os.Lstat(root); os.IsNotExist(err) && top != ManifestFile {
			continue
		}
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(dir, p)
			if err != nil {
				return err
			}
			name := filepath.ToSlash(rel)
			switch {
			case d.IsDir():
				entries = append(entries, packEntry{name: name + "/", path: p, dir: true})
			case d.Type().IsRegular():
				entries = append(entries, packEntry{name: name, path: p})
			default:
				return fmt.Errorf("%s: a bundle holds only regular files and directories", p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	return entries, nil
}

func writeEntry(tw *tar.Writer, e packEntry, created time.Time) error {
	h := &tar.Header{Name: e.name, ModTime: created, Format: tar.FormatUSTAR}
	var body []byte
	if e.dir {
		h.Typeflag, h.Mode = tar.TypeDir, 0o755
	} else {
		var err error
		if body, err = os.ReadFile(e.path); err != nil {
			return err
		}
		h.Typeflag, h.Mode, h.Size = tar.TypeReg, 0o644, int64(len(body))
	}
	if err := tw.WriteHeader(h); err != nil {
		return fmt.Errorf("packing %s: %w", e.name, err)
	}
	_, err := tw.Write(body)
	return err
}

// CheckLimits refuses a tree a pull would refuse: more entries or file
// bytes than lim allows.
func CheckLimits(dir string, lim Limits) error {
	entries, err := packEntries(dir)
	if err != nil {
		return err
	}
	if len(entries) > lim.Entries {
		return fmt.Errorf("%s holds %d entries, more than the %d a bundle may hold", dir, len(entries), lim.Entries)
	}
	var total int64
	for _, e := range entries {
		if e.dir {
			continue
		}
		st, err := os.Stat(e.path)
		if err != nil {
			return err
		}
		total += st.Size()
	}
	if total > lim.Bytes {
		return fmt.Errorf("%s holds %d bytes of files, more than the %d a bundle may hold", dir, total, lim.Bytes)
	}
	return nil
}

// Limits bound what Unpack accepts.
type Limits struct {
	Entries int   // most tar entries
	Bytes   int64 // most uncompressed file bytes
}

// DefaultLimits are the bundle format's limits.
var DefaultLimits = Limits{Entries: 10000, Bytes: 64 << 20}

// MaxLayerSize is the largest layer descriptor pull fetches.
const MaxLayerSize = 32 << 20

// Unpack extracts a bundle layer into dest, which must not exist. It
// refuses, writing nothing, an absolute path, a ".." element, a symlink, a
// hard link, a device or FIFO, a duplicate path, a top-level entry other
// than manifest.json, content/ and data/, more entries or bytes than the
// limits allow, and a tree whose manifest.json does not validate or does
// not list exactly the files present. It returns the manifest.
func Unpack(r io.Reader, dest string, lim Limits) (*Manifest, error) {
	if _, err := os.Lstat(dest); err == nil {
		return nil, fmt.Errorf("unpacking into %s: it already exists", dest)
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dest), ".unpack-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp) // nothing is left when a check fails; after the rename it is gone
	if err := extract(r, tmp, lim); err != nil {
		return nil, err
	}
	m, err := Read(tmp)
	if err != nil {
		return nil, err
	}
	if err := CheckTree(tmp, m); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, dest); err != nil {
		return nil, err
	}
	return m, nil
}

func extract(r io.Reader, dir string, lim Limits) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("reading the layer: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	seen := map[string]bool{}
	var n int
	var total int64
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading the layer: %w", err)
		}
		n++
		if n > lim.Entries {
			return fmt.Errorf("the layer holds more than %d entries", lim.Entries)
		}
		name, err := checkEntry(h)
		if err != nil {
			return err
		}
		if seen[name] {
			return fmt.Errorf("%s: duplicate entry", h.Name)
		}
		seen[name] = true
		target := filepath.Join(dir, filepath.FromSlash(name))
		if h.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, 0o755); err != nil { //nolint:gosec // published content
				return err
			}
			continue
		}
		total += h.Size
		if h.Size < 0 || total > lim.Bytes {
			return fmt.Errorf("the layer holds more than %d bytes", lim.Bytes)
		}
		if err := writeFile(target, tr, h.Size); err != nil {
			return err
		}
	}
}

// checkEntry refuses every entry a bundle may not hold and returns the
// cleaned slash-separated name.
func checkEntry(h *tar.Header) (string, error) {
	if err := checkType(h); err != nil {
		return "", err
	}
	name := h.Name
	if name == "" || name[0] == '/' || filepath.IsAbs(name) {
		return "", fmt.Errorf("%s: absolute path", h.Name)
	}
	trimmed := trimSlash(name)
	for _, el := range splitSlash(trimmed) {
		if el == ".." || el == "." || el == "" {
			return "", fmt.Errorf("%s: path element %q", h.Name, el)
		}
	}
	top, _, _ := cut(trimmed)
	isManifest := trimmed == ManifestFile && h.Typeflag == tar.TypeReg
	if !isManifest && top != ContentDir && top != DataDir {
		return "", fmt.Errorf("%s: not allowed at the top level of a bundle (only manifest.json, content/ and data/)", h.Name)
	}
	return trimmed, nil
}

func checkType(h *tar.Header) error {
	switch h.Typeflag {
	case tar.TypeReg, tar.TypeDir:
		return nil
	case tar.TypeSymlink:
		return fmt.Errorf("%s: symlink; a bundle holds only regular files and directories", h.Name)
	case tar.TypeLink:
		return fmt.Errorf("%s: hard link; a bundle holds only regular files and directories", h.Name)
	default:
		return fmt.Errorf("%s: entry type %q; a bundle holds only regular files and directories", h.Name, h.Typeflag)
	}
}

func writeFile(target string, r io.Reader, size int64) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { //nolint:gosec // published content
		return err
	}
	f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644) //nolint:gosec // the name passed checkEntry
	if err != nil {
		return err
	}
	if _, err := io.CopyN(f, r, size); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
