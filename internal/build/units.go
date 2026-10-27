package build

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/open-platform-model/docs-kit/internal/config"
)

// units reads each configured unit's version from its JSON file in the source
// tree and records them in the manifest. A missing file, a missing key and a
// value that is not an exact version without "v" all refuse the build.
func (s *assembly) units(units map[string]config.Unit) error {
	if len(units) == 0 {
		return nil
	}
	root, err := os.OpenRoot(s.unitsDir())
	if err != nil {
		return fmt.Errorf("%s: units: %w", s.m.Project, err)
	}
	defer func() { _ = root.Close() }() // read-only
	names := make([]string, 0, len(units))
	for n := range units {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make(map[string]string, len(units))
	for _, n := range names {
		u := units[n]
		raw, err := root.ReadFile(u.File)
		if err != nil {
			return fmt.Errorf("%s: unit %s: cannot read %s from the source tree: %w; units.%s.file in %s names a JSON file such as .release-please-manifest.json", s.m.Project, n, u.File, err, n, s.cfgPath)
		}
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(raw, &doc); err != nil {
			return fmt.Errorf("%s: unit %s: %s is not a JSON object: %w", s.m.Project, n, u.File, err)
		}
		v, ok := doc[u.Key]
		if !ok {
			return fmt.Errorf("%s: unit %s: %s has no key %q; units.%s.key in %s names a top-level key of that file", s.m.Project, n, u.File, u.Key, n, s.cfgPath)
		}
		var ver string
		if err := json.Unmarshal(v, &ver); err != nil || !reSemVer.MatchString(ver) {
			return fmt.Errorf("%s: unit %s: %s key %q holds %s, not an exact version such as 0.1.1 (a string, no v)", s.m.Project, n, u.File, u.Key, v)
		}
		out[n] = ver
	}
	s.m.Units = out
	return nil
}

// unitsDir is the tree the unit files are read from: the source tree.
func (s *assembly) unitsDir() string { return s.o.Source }
