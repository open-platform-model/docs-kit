// Package bundle is the docs bundle on disk and on the wire: the tree of
// manifest.json, content/ and data/, its manifest, the OCI media types and
// annotations, deterministic packing and guarded unpacking.
package bundle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/open-platform-model/docs-kit/schema"
)

// Identifiers of the bundle format.
const (
	SchemaID     = "docs.opmodel.dev/bundle/v1"
	ArtifactType = "application/vnd.opmodel.docs.bundle.v1"
	LayerType    = "application/vnd.opmodel.docs.bundle.layer.v1.tar+gzip"
	ManifestFile = "manifest.json"
	ContentDir   = "content"
	DataDir      = "data"
)

// Annotation keys on the pushed manifest.
const (
	AnnVersion  = "org.opencontainers.image.version"
	AnnRevision = "org.opencontainers.image.revision"
	AnnSource   = "org.opencontainers.image.source"
	AnnCreated  = "org.opencontainers.image.created"
	AnnProject  = "dev.opmodel.docs.project"
	AnnDocsRev  = "dev.opmodel.docs.revision"
	AnnDialect  = "dev.opmodel.docs.dialect"
	AnnTool     = "dev.opmodel.docs.tool"
)

// Source is the commit a bundle was built from.
type Source struct {
	Repo    string   `json:"repo"`
	Commit  string   `json:"commit"`
	Ref     string   `json:"ref"`
	Dirty   bool     `json:"dirty,omitempty"`
	Patches []string `json:"patches,omitempty"`
}

// Placement is where the site mounts content/.
type Placement struct {
	Kind string `json:"kind"`
	Root string `json:"root"`
}

// Page is one file under content/.
type Page struct {
	Path      string `json:"path"`
	Source    string `json:"source,omitempty"`
	Lastmod   string `json:"lastmod,omitempty"`
	Generated bool   `json:"generated"`
}

// DataFile is one file under data/.
type DataFile struct {
	Path   string `json:"path"`
	Schema string `json:"schema"`
}

// Manifest is manifest.json. Field order is the file's key order.
type Manifest struct {
	Schema    string     `json:"schema"`
	Project   string     `json:"project"`
	Version   string     `json:"version"`
	Revision  int        `json:"revision"`
	Source    Source     `json:"source"`
	Created   string     `json:"created,omitempty"`
	Tool      string     `json:"tool"`
	Dialect   int        `json:"dialect"`
	Placement Placement  `json:"placement"`
	Pages     []Page     `json:"pages"`
	Data      []DataFile `json:"data"`
}

// Segment is the URL segment the bundle is shown under: "MAJOR.MINOR" of
// its version, or "edge".
func (m *Manifest) Segment() string {
	if m.Version == "edge" {
		return "edge"
	}
	parts := strings.SplitN(m.Version, ".", 3)
	if len(parts) < 2 {
		return m.Version
	}
	return parts[0] + "." + parts[1]
}

// SourceURL is the repository URL the source annotation names.
func (m *Manifest) SourceURL() string {
	return "https://github.com/" + m.Source.Repo
}

// Annotations are the manifest annotations of a pushed bundle, each equal
// to the matching manifest.json field.
func (m *Manifest) Annotations() map[string]string {
	return map[string]string{
		AnnVersion:  m.Version,
		AnnRevision: m.Source.Commit,
		AnnSource:   m.SourceURL(),
		AnnCreated:  m.Created,
		AnnProject:  m.Project,
		AnnDocsRev:  strconv.Itoa(m.Revision),
		AnnDialect:  strconv.Itoa(m.Dialect),
		AnnTool:     m.Tool,
	}
}

// Encode serializes the manifest: two-space indent, trailing newline.
func (m *Manifest) Encode() ([]byte, error) {
	c := *m
	if c.Pages == nil {
		c.Pages = []Page{}
	}
	if c.Data == nil {
		c.Data = []DataFile{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&c); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Parse validates manifest.json bytes against #Manifest and decodes them.
// name is the file the bytes came from, for error positions.
func Parse(name string, data []byte) (*Manifest, error) {
	if _, err := schema.ValidateJSON("#Manifest", name, data); err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return &m, nil
}

// Read reads and validates <dir>/manifest.json.
func Read(dir string) (*Manifest, error) {
	p := filepath.Join(dir, ManifestFile)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", p, err)
	}
	return Parse(p, data)
}

// Write validates m and writes it to <dir>/manifest.json.
func Write(dir string, m *Manifest) error {
	data, err := m.Encode()
	if err != nil {
		return err
	}
	p := filepath.Join(dir, ManifestFile)
	if _, err := Parse(p, data); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644) //nolint:gosec // a bundle is published content
}

// CheckTree checks a bundle directory: only manifest.json, content/ and
// data/ at the top level, only regular files and directories below, and
// manifest.json's pages and data listing exactly the files present.
func CheckTree(dir string, m *Manifest) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", dir, err)
	}
	var problems []string
	for _, e := range entries {
		switch e.Name() {
		case ManifestFile, ContentDir, DataDir:
		default:
			problems = append(problems, fmt.Sprintf("%s: not allowed at the top level of a bundle (only manifest.json, content/ and data/)", e.Name()))
		}
	}
	pages, err := listFiles(filepath.Join(dir, ContentDir))
	if err != nil {
		return err
	}
	data, err := listFiles(filepath.Join(dir, DataDir))
	if err != nil {
		return err
	}
	listedPages := map[string]bool{}
	for _, p := range m.Pages {
		listedPages[p.Path] = true
	}
	listedData := map[string]bool{}
	for _, d := range m.Data {
		listedData[d.Path] = true
	}
	problems = append(problems, compare(ContentDir, pages, listedPages)...)
	problems = append(problems, compare(DataDir, data, listedData)...)
	if len(problems) > 0 {
		return &TreeError{Dir: dir, Problems: problems}
	}
	return nil
}

// TreeError lists what is wrong with a bundle tree.
type TreeError struct {
	Dir      string
	Problems []string
}

func (e *TreeError) Error() string {
	return e.Dir + ": " + strings.Join(e.Problems, "; ")
}

func compare(sub string, present []string, listed map[string]bool) []string {
	var out []string
	seen := map[string]bool{}
	for _, p := range present {
		seen[p] = true
		if !listed[p] {
			out = append(out, fmt.Sprintf("%s/%s is not listed in manifest.json", sub, p))
		}
	}
	var missing []string
	for p := range listed {
		if !seen[p] {
			missing = append(missing, p)
		}
	}
	sort.Strings(missing)
	for _, p := range missing {
		out = append(out, fmt.Sprintf("manifest.json lists %s/%s, which does not exist", sub, p))
	}
	return out
}

// listFiles lists the regular files under dir, slash-separated and
// relative to it, in byte order. A missing dir has none; anything other
// than a regular file or a directory is an error.
func listFiles(dir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && p == dir {
				return filepath.SkipDir
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s: not a regular file", p)
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(out)
	return out, err
}
