package pull

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	sha  = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	sha2 = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	cmt  = "0123456789abcdef0123456789abcdef01234567"
)

func docsEntry(site, project, role string, local bool) DocsEntry {
	e := DocsEntry{Site: site, Project: project, Role: role, Local: local, Version: "1.0.0", Commit: cmt, Dialect: 1, BuiltBy: "0.4.0", Dir: "_versions/" + site + "/" + project}
	if !local {
		e.Tag, e.Repository, e.Digest = "1.0.0", "ghcr.io/open-platform-model/docs/"+project, sha2
		e.Signer = &Signer{Workflow: "w", Repository: "https://github.com/open-platform-model/" + project, Ref: "refs/heads/main"}
	}
	return e
}

// A lock without docs, as every lock before site versions, still validates
// and gains no docs key.
func TestLockWithoutDocs(t *testing.T) {
	l := &Lock{Schema: LockSchema, Tool: "0.4.0", Config: sha}
	b, err := l.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"docs"`) {
		t.Fatalf("lock:\n%s", b)
	}
}

func TestDocsLockRoundTripAndOrder(t *testing.T) {
	anchor := docsEntry("v1.0", "cli", RoleAnchor, true)
	anchor.Pins = map[string]string{"library": "1.0.0-beta.1", "core": "2.0.0-beta.1"}
	l := &Lock{Schema: LockSchema, Tool: "0.4.0", Config: sha, Docs: []DocsEntry{
		docsEntry("v1.0", "opm", RoleTag, false),
		docsEntry("v1.0", "library", RolePinned, false),
		docsEntry("v0.10", "cli", RoleAnchor, false),
		docsEntry("v1.0", "core", RolePinned, false),
		anchor,
		docsEntry("v0.9", "cli", RoleAnchor, false),
	}}
	b, err := l.Encode()
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "lock.json")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := ReadLock(p)
	if err != nil {
		t.Fatal(err)
	}
	order := make([]string, 0, len(r.Docs))
	for _, e := range r.Docs {
		order = append(order, e.Site+"/"+e.Role+"/"+e.Project)
	}
	if got := strings.Join(order, ","); got != "v0.9/anchor/cli,v0.10/anchor/cli,v1.0/anchor/cli,v1.0/pinned/core,v1.0/pinned/library,v1.0/tag/opm" {
		t.Fatalf("order %s", got)
	}
	if a := r.Docs[2]; !a.Local || a.Digest != "" || a.Pins["core"] != "2.0.0-beta.1" {
		t.Fatalf("anchor %+v", a)
	}
	again, err := r.Encode()
	if err != nil || !bytes.Equal(again, b) {
		t.Fatalf("round trip differs (%v):\n%s\nwant:\n%s", err, again, b)
	}
	// Key order: a local entry has "local" after "role" and "pins" before
	// "dir"; a pulled one "tag" after "role" and "signer" before "dir".
	for _, want := range []string{
		"\"role\": \"anchor\",\n      \"local\": true,\n      \"version\"",
		"\"builtBy\": \"0.4.0\",\n      \"pins\": {\n        \"core\": \"2.0.0-beta.1\",\n        \"library\": \"1.0.0-beta.1\"\n      },\n      \"dir\"",
		"\"role\": \"pinned\",\n      \"tag\": \"1.0.0\",\n      \"repository\"",
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("lock lacks %q:\n%s", want, b)
		}
	}
}

func TestDocsLockRefusesABadEntry(t *testing.T) {
	bad := docsEntry("v1.0", "cli", "pin", false)
	l := &Lock{Schema: LockSchema, Tool: "0.4.0", Config: sha, Docs: []DocsEntry{bad}}
	if _, err := l.Encode(); err == nil {
		t.Fatal("a lock with role pin validated")
	}
}
