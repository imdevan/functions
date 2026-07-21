package config

import "github.com/functions/internal/domain"

// Load returns the config for cwd. Uses overridePath when non-empty; falls back
// to DefaultConfig on error.
func Load(cwd, overridePath string) domain.Config {
	m := NewManager(cwd)
	var (
		cfg domain.Config
		err error
	)
	if overridePath != "" {
		cfg, err = m.LoadWithOverride(overridePath)
	} else {
		cfg, err = m.Load()
	}
	if err != nil {
		return domain.DefaultConfig()
	}
	return cfg
}
