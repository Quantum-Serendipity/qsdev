package config

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Migration describes a single schema migration step from one version to another.
type Migration struct {
	FromVersion int
	ToVersion   int
	Description string
	Migrate     func(raw map[string]any) (map[string]any, error)
}

// MigrationChain is the ordered list of migrations. Currently empty because
// only v1 exists; future schema changes will add entries here.
var MigrationChain = []Migration{}

// MigrateConfig applies all necessary migrations to bring raw config data
// from fromVersion to the current schema version. Returns the migrated raw
// map and any error encountered.
func MigrateConfig(raw map[string]any, fromVersion int) (map[string]any, error) {
	if fromVersion == types.ConfigVersionCurrent {
		return raw, nil
	}

	if fromVersion > types.ConfigVersionCurrent {
		return nil, fmt.Errorf(
			"config version %d is newer than this binary supports (max %d); please update qsdev",
			fromVersion, types.ConfigVersionCurrent)
	}

	if fromVersion < types.ConfigVersionMin {
		return nil, fmt.Errorf(
			"config version %d is too old to migrate (minimum %d)",
			fromVersion, types.ConfigVersionMin)
	}

	current := fromVersion
	for _, m := range MigrationChain {
		if m.FromVersion != current {
			continue
		}

		migrated, err := m.Migrate(raw)
		if err != nil {
			return nil, fmt.Errorf("migration v%d -> v%d (%s): %w",
				m.FromVersion, m.ToVersion, m.Description, err)
		}

		raw = migrated
		current = m.ToVersion
	}

	if current != types.ConfigVersionCurrent {
		return nil, fmt.Errorf(
			"migration chain incomplete: reached version %d but current is %d",
			current, types.ConfigVersionCurrent)
	}

	// Update the version field in the raw map.
	raw["version"] = types.ConfigVersionCurrent

	return raw, nil
}

// NeedsMigration returns true if configVersion is older than the current
// schema version and a migration path exists.
func NeedsMigration(configVersion int) bool {
	return configVersion > 0 && configVersion < types.ConfigVersionCurrent
}
