package build

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/open-platform-model/docs-kit/internal/config"
)

// units records each unit's version in the manifest: the value Options.Units
// gives for it, else the value read from its JSON file in the source tree.
// The units are those the config declares plus those Options.Units names, so
// a revision of a release whose own config declares none still records the
// units a refresh read from main.
func (s *assembly) units(units map[string]config.Unit) error {
	if len(units) == 0 && len(s.o.Units) == 0 {
		return nil
	}
	read := map[string]config.Unit{}
	for n, u := range units {
		if _, ok := s.o.Units[n]; !ok {
			read[n] = u
		}
	}
	got, err := ReadUnits(s.o.Source, s.m.Project, s.cfgPath, read)
	if err != nil {
		return err
	}
	for n, v := range s.o.Units {
		got[n] = v
	}
	s.m.Units = got
	return nil
}

// ReadUnits reads each unit's version from its JSON file under dir. A
// missing file, a missing key and a value that is not an exact version
// without "v" all refuse, naming the file and the key. project and cfgPath
// only name the source of the unit in messages.
func ReadUnits(dir, project, cfgPath string, units map[string]config.Unit) (map[string]string, error) {
	out := make(map[string]string, len(units))
	if len(units) == 0 {
		return out, nil
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("%s: units: %w", project, err)
	}
	defer func() { _ = root.Close() }() // read-only
	names := make([]string, 0, len(units))
	for n := range units {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		u := units[n]
		raw, err := root.ReadFile(u.File)
		if err != nil {
			return nil, fmt.Errorf("%s: unit %s: cannot read %s from the source tree: %w; units.%s.file in %s names a JSON file such as .release-please-manifest.json", project, n, u.File, err, n, cfgPath)
		}
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("%s: unit %s: %s is not a JSON object: %w", project, n, u.File, err)
		}
		v, ok := doc[u.Key]
		if !ok {
			return nil, fmt.Errorf("%s: unit %s: %s has no key %q; units.%s.key in %s names a top-level key of that file", project, n, u.File, u.Key, n, cfgPath)
		}
		var ver string
		if err := json.Unmarshal(v, &ver); err != nil || !reSemVer.MatchString(ver) {
			return nil, fmt.Errorf("%s: unit %s: %s key %q holds %s, not an exact version such as 0.1.1 (a string, no v)", project, n, u.File, u.Key, v)
		}
		out[n] = ver
	}
	return out, nil
}
