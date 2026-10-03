package build

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/open-platform-model/docs-kit/internal/config"
)

// PinsSchema is the schema id of the document a pins command prints.
const PinsSchema = "docs.opmodel.dev/pins/v1"

// reSemVer is the manifest's #SemVer: no "v".
var reSemVer = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

// pins runs the config's pins command and records its pins in the
// manifest: exactly the configured projects, each an exact version.
func (s *assembly) pins(p *config.Pins) error {
	if p == nil {
		return nil
	}
	out, err := s.commands.Run(s.ctx, p.Command, PinsSchema)
	if err != nil {
		return err
	}
	var doc struct {
		Schema string            `json:"schema"`
		Pins   map[string]string `json:"pins"`
	}
	dec := json.NewDecoder(strings.NewReader(string(out)))
	dec.DisallowUnknownFields()
	argv := strings.Join(p.Command, " ")
	if err := dec.Decode(&doc); err != nil {
		return fmt.Errorf("%s: pins command `%s`: not a pins document: %w", s.m.Project, argv, err)
	}
	var missing, extra []string
	for _, proj := range p.Projects {
		if _, ok := doc.Pins[proj]; !ok {
			missing = append(missing, proj)
		}
	}
	for proj, v := range doc.Pins {
		if !slices.Contains(p.Projects, proj) {
			extra = append(extra, proj)
			continue
		}
		if !reSemVer.MatchString(v) {
			return fmt.Errorf("%s: pins command `%s`: %s pins %q, not an exact version such as 1.0.0-beta.1 (no v)", s.m.Project, argv, proj, v)
		}
	}
	sort.Strings(extra)
	switch {
	case len(missing) > 0:
		return fmt.Errorf("%s: pins command `%s` printed no pin for %s; pins.projects in %s lists %s", s.m.Project, argv, strings.Join(missing, ", "), s.cfgPath, strings.Join(p.Projects, ", "))
	case len(extra) > 0:
		return fmt.Errorf("%s: pins command `%s` pins %s, which pins.projects in %s does not list", s.m.Project, argv, strings.Join(extra, ", "), s.cfgPath)
	}
	s.m.Pins = doc.Pins
	return nil
}
