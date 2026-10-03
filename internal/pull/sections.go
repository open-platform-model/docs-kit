package pull

import (
	"fmt"
	"sort"

	"github.com/open-platform-model/docs-kit/internal/config"
	"github.com/open-platform-model/docs-kit/internal/tags"
)

// placed is a project pulled by segments into <out>/<project>/<segment>/:
// a tab, or a section, which has only its edge segment.
type placed struct {
	config.Tab
	section bool
}

// kind is the placement kind its bundles must carry.
func (pl placed) kind() string {
	if pl.section {
		return config.PlacementSection
	}
	return "tab"
}

// placements lists every tab and every section of the config. A section
// is a tab that shows only edge.
func placements(cfg *config.Pull) map[string]placed {
	out := make(map[string]placed, len(cfg.Tabs)+len(cfg.Sections))
	for name, t := range cfg.Tabs {
		out[name] = placed{Tab: t}
	}
	for name, s := range cfg.Sections {
		out[name] = placed{Tab: config.Tab{Repo: s.Repo, Root: s.Root, Edge: true}, section: true}
	}
	return out
}

// placedProjects returns the tabs and sections in name order.
func (p *puller) placedProjects() []string {
	out := make([]string, 0, len(p.places))
	for name := range p.places {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// sectionSegments picks a section's one segment: edge, which must exist,
// since an empty section cannot be shown.
func sectionSegments(project string, all []string) ([]string, error) {
	for _, t := range all {
		if t == tags.Edge {
			return []string{tags.Edge}, nil
		}
	}
	return nil, fmt.Errorf("%s has no edge build; the section would be empty, so publish main first", project)
}
