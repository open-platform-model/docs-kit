package serve

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/docs-kit/internal/config"
)

// sourcePaths names, per source kind, the options holding the paths its
// build reads. A kind not listed (cobra runs a repository command; a kind
// this table does not know yet) watches the whole source tree.
var sourcePaths = map[string][]string{
	"cue-catalog":     {"module"},
	"cue-definitions": {"package"},
	"crd":             {"dir", "samples"},
}

// watchRoots lists the files and directories whose change rebuilds one
// bundle: the config, each source's paths, or the whole source tree when
// a source or the pins run a repository command (C14), whose inputs are
// the tree's files.
func watchRoots(source, cfgPath string, b config.Bundle) []string {
	roots := []string{abs(cfgPath)}
	whole := b.Pins != nil
	for _, src := range b.Sources {
		if src.Markdown != nil {
			roots = append(roots, abs(filepath.Join(source, filepath.FromSlash(src.Markdown.Dir))))
			continue
		}
		keys, ok := sourcePaths[src.Kind]
		if !ok {
			whole = true
			continue
		}
		for _, k := range keys {
			if p, err := src.Value.LookupPath(cue.ParsePath(k)).String(); err == nil && p != "" {
				roots = append(roots, abs(filepath.Join(source, filepath.FromSlash(p))))
			}
		}
	}
	if whole {
		roots = append(roots, abs(source))
	}
	sort.Strings(roots)
	return roots
}

func abs(p string) string {
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return p
}

// stamp is what a poll compares: a file's size and modification time.
type stamp struct {
	size int64
	mod  time.Time
}

// watcher polls each project's roots and reports the projects with a file
// added, removed or changed since the last poll. Directories whose name
// starts with "." (.git, .cue-cache) are skipped.
type watcher struct {
	roots map[string][]string // project -> roots
	last  map[string]map[string]stamp
}

func newWatcher(roots map[string][]string) *watcher {
	w := &watcher{roots: roots, last: map[string]map[string]stamp{}}
	for p := range roots {
		w.last[p] = snapshot(roots[p])
	}
	return w
}

// setRoots replaces the roots, taking a new baseline for every project
// whose roots changed.
func (w *watcher) setRoots(roots map[string][]string) {
	for p, r := range roots {
		if strings.Join(r, "\x00") != strings.Join(w.roots[p], "\x00") {
			w.last[p] = snapshot(r)
		}
	}
	for p := range w.last {
		if _, ok := roots[p]; !ok {
			delete(w.last, p)
		}
	}
	w.roots = roots
}

// poll returns, sorted, the projects whose files changed since the last
// poll.
func (w *watcher) poll() []string {
	var out []string
	for p, r := range w.roots {
		now := snapshot(r)
		if !same(now, w.last[p]) {
			out = append(out, p)
		}
		w.last[p] = now
	}
	sort.Strings(out)
	return out
}

func same(a, b map[string]stamp) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if u, ok := b[k]; !ok || u.size != v.size || !u.mod.Equal(v.mod) {
			return false
		}
	}
	return true
}

// snapshot stamps every regular file under roots. A missing root holds
// nothing, so creating it counts as a change.
func snapshot(roots []string) map[string]stamp {
	out := map[string]stamp{}
	for _, r := range roots {
		_ = filepath.WalkDir(r, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil //nolint:nilerr // a file removed during the walk is seen by the next poll
			}
			if d.IsDir() {
				if p != r && strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil //nolint:nilerr // as above
			}
			out[p] = stamp{size: info.Size(), mod: info.ModTime()}
			return nil
		})
	}
	return out
}
