package pull

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/tags"
	"github.com/open-platform-model/docs-kit/schema"
)

// LockSchema identifies the lock format.
const LockSchema = "docs.opmodel.dev/lock/v1"

// Signer is who signed a pulled bundle.
type Signer struct {
	Workflow   string `json:"workflow"`
	Repository string `json:"repository"`
	Ref        string `json:"ref"`
}

// Entry is one locked bundle. Field order is the lock's key order; a local
// entry leaves out Tag, Repository, Digest and Signer.
type Entry struct {
	Project    string  `json:"project"`
	Root       string  `json:"root"`
	Segment    string  `json:"segment"`
	Local      bool    `json:"-"`
	Tag        string  `json:"tag,omitempty"`
	Repository string  `json:"repository,omitempty"`
	Digest     string  `json:"digest,omitempty"`
	Version    string  `json:"version"`
	Revision   int     `json:"revision"`
	Commit     string  `json:"commit"`
	Dialect    int     `json:"dialect"`
	BuiltBy    string  `json:"builtBy"`
	Signer     *Signer `json:"signer,omitempty"`
	Dir        string  `json:"dir"`
}

// localEntry is a local entry's serialized form, with "local" after
// "segment".
type localEntry struct {
	Project  string `json:"project"`
	Root     string `json:"root"`
	Segment  string `json:"segment"`
	Local    bool   `json:"local"`
	Version  string `json:"version"`
	Revision int    `json:"revision"`
	Commit   string `json:"commit"`
	Dialect  int    `json:"dialect"`
	BuiltBy  string `json:"builtBy"`
	Dir      string `json:"dir"`
}

// MarshalJSON writes a pulled or a local entry in its key order.
func (e Entry) MarshalJSON() ([]byte, error) {
	if e.Local {
		return json.Marshal(localEntry{e.Project, e.Root, e.Segment, true, e.Version, e.Revision, e.Commit, e.Dialect, e.BuiltBy, e.Dir})
	}
	type plain Entry
	return json.Marshal(plain(e))
}

// UnmarshalJSON reads either form.
func (e *Entry) UnmarshalJSON(b []byte) error {
	type plain Entry
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	var l struct {
		Local bool `json:"local"`
	}
	if err := json.Unmarshal(b, &l); err != nil {
		return err
	}
	*e = Entry(p)
	e.Local = l.Local
	return nil
}

// Lock is the lock file.
type Lock struct {
	Schema  string  `json:"schema"`
	Tool    string  `json:"tool"`
	Config  string  `json:"config"`
	Bundles []Entry `json:"bundles"`
	// History records each history.json this pull wrote, so the site can
	// check the file it mounts; left out when none was written.
	History []HistoryEntry `json:"history,omitempty"`
	// Docs records each site version's docs bundles; left out when the
	// config has no versions.
	Docs []DocsEntry `json:"docs,omitempty"`
}

// Roles of a docs bundle in its site version, in lock order.
const (
	RoleAnchor = "anchor"
	RolePinned = "pinned"
	RoleTag    = "tag"
)

var roleOrder = map[string]int{RoleAnchor: 0, RolePinned: 1, RoleTag: 2}

// DocsEntry is one locked docs bundle of a site version. Field order is the
// lock's key order; a local entry leaves out Tag, Repository, Digest and
// Signer.
type DocsEntry struct {
	Site       string            `json:"site"`
	Project    string            `json:"project"`
	Role       string            `json:"role"`
	Local      bool              `json:"-"`
	Tag        string            `json:"tag,omitempty"`
	Repository string            `json:"repository,omitempty"`
	Digest     string            `json:"digest,omitempty"`
	Version    string            `json:"version"`
	Revision   int               `json:"revision"`
	Commit     string            `json:"commit"`
	Dialect    int               `json:"dialect"`
	BuiltBy    string            `json:"builtBy"`
	Signer     *Signer           `json:"signer,omitempty"`
	Pins       map[string]string `json:"pins,omitempty"`
	Dir        string            `json:"dir"`
}

// localDocsEntry is a local docs entry's serialized form, with "local"
// after "role".
type localDocsEntry struct {
	Site     string            `json:"site"`
	Project  string            `json:"project"`
	Role     string            `json:"role"`
	Local    bool              `json:"local"`
	Version  string            `json:"version"`
	Revision int               `json:"revision"`
	Commit   string            `json:"commit"`
	Dialect  int               `json:"dialect"`
	BuiltBy  string            `json:"builtBy"`
	Pins     map[string]string `json:"pins,omitempty"`
	Dir      string            `json:"dir"`
}

// MarshalJSON writes a pulled or a local docs entry in its key order.
func (e DocsEntry) MarshalJSON() ([]byte, error) {
	if e.Local {
		return json.Marshal(localDocsEntry{e.Site, e.Project, e.Role, true, e.Version, e.Revision, e.Commit, e.Dialect, e.BuiltBy, e.Pins, e.Dir})
	}
	type plain DocsEntry
	return json.Marshal(plain(e))
}

// UnmarshalJSON reads either form.
func (e *DocsEntry) UnmarshalJSON(b []byte) error {
	type plain DocsEntry
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	var l struct {
		Local bool `json:"local"`
	}
	if err := json.Unmarshal(b, &l); err != nil {
		return err
	}
	*e = DocsEntry(p)
	e.Local = l.Local
	return nil
}

// sortDocs orders docs entries by site version (numeric MAJOR, then
// MINOR), then role (anchor, pinned, tag), then project.
func sortDocs(es []DocsEntry) {
	sort.SliceStable(es, func(i, j int) bool {
		a, b := &es[i], &es[j]
		if c := config.CompareSiteVersions(a.Site, b.Site); c != 0 {
			return c < 0
		}
		if a.Role != b.Role {
			return roleOrder[a.Role] < roleOrder[b.Role]
		}
		return a.Project < b.Project
	})
}

// HistoryEntry is one tab's history.json: its SHA-256 and its path
// relative to the lock's directory.
type HistoryEntry struct {
	Project string `json:"project"`
	Digest  string `json:"digest"`
	Path    string `json:"path"`
}

// sortEntries orders entries by project, then segment: minors by number,
// edge last.
func sortEntries(es []Entry) {
	sort.SliceStable(es, func(i, j int) bool {
		if es[i].Project != es[j].Project {
			return es[i].Project < es[j].Project
		}
		return tags.CompareMinor(es[i].Segment, es[j].Segment) < 0
	})
}

// Encode serializes the lock: two-space indent, trailing newline, entries
// sorted (bundles by project and segment, history by project, docs by site
// version, role and project), no timestamps. It validates the result
// against #Lock.
func (l *Lock) Encode() ([]byte, error) {
	c := *l
	c.Bundles = append([]Entry{}, l.Bundles...)
	if c.Bundles == nil {
		c.Bundles = []Entry{}
	}
	sortEntries(c.Bundles)
	c.History = append([]HistoryEntry(nil), l.History...)
	sort.SliceStable(c.History, func(i, j int) bool { return c.History[i].Project < c.History[j].Project })
	c.Docs = append([]DocsEntry(nil), l.Docs...)
	sortDocs(c.Docs)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&c); err != nil {
		return nil, err
	}
	if _, err := schema.ValidateJSON("#Lock", "lock.json", buf.Bytes()); err != nil {
		return nil, fmt.Errorf("the lock does not validate: %w", err)
	}
	return buf.Bytes(), nil
}

// ReadLock reads and validates a lock file.
func ReadLock(path string) (*Lock, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading the lock %s: %w", path, err)
	}
	if _, err := schema.ValidateJSON("#Lock", path, b); err != nil {
		return nil, err
	}
	var l Lock
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &l, nil
}
