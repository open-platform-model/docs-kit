package pull

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"

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
// sorted, no timestamps. It validates the result against #Lock.
func (l *Lock) Encode() ([]byte, error) {
	c := *l
	c.Bundles = append([]Entry{}, l.Bundles...)
	if c.Bundles == nil {
		c.Bundles = []Entry{}
	}
	sortEntries(c.Bundles)
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
