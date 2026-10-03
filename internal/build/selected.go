package build

import (
	"github.com/open-platform-model/docs-kit/internal/config"
)

// Selected loads the config Run would read for o and returns it with the
// projects Run would build, in build order, refusing what Run refuses
// (a missing or invalid config, an unknown --project) with a UsageError.
func Selected(o Options) (*config.Config, []string, error) {
	cfgPath, _, err := configPath(o)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, nil, &UsageError{err}
	}
	projects, err := selectProjects(cfg, o.Projects)
	if err != nil {
		return nil, nil, err
	}
	return cfg, projects, nil
}
