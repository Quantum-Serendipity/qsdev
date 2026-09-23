package config

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Migration describes a single schema migration step from one version to another.
type Migration struct {
	FromVersion int
	ToVersion   int
	Description string
	Migrate     func(raw map[string]any) (map[string]any, error)
}

// migrationChain is the ordered list of migrations. Currently empty because
// only v1 exists; future schema changes will add entries here. It is
// unexported so importers cannot alter the migration path.
var migrationChain = []Migration{}

// ParseConfigVersion converts a YAML-decoded "version" value to an int.
// It reports false when the value is not a whole number.
func ParseConfigVersion(v any) (int, bool) {
	return toInt(v)
}

// CheckMigration reports whether a config at configVersion must be migrated
// to reach the current schema version. It returns an error when the version
// cannot be migrated at all: newer than this binary supports, or older than
// the minimum supported version (which includes zero and negative values).
// needed is false with a nil error only when the version is current.
func CheckMigration(configVersion int) (needed bool, err error) {
	switch {
	case configVersion > types.ConfigVersionCurrent:
		return false, fmt.Errorf(
			"config version %d is newer than this binary supports (max %d); please update %s",
			configVersion, types.ConfigVersionCurrent, branding.Get().AppName)
	case configVersion < types.ConfigVersionMin:
		return false, fmt.Errorf(
			"config version %d is too old to migrate (minimum %d)",
			configVersion, types.ConfigVersionMin)
	default:
		return configVersion < types.ConfigVersionCurrent, nil
	}
}

// MigrateConfig applies all necessary migrations to bring raw config data
// from fromVersion to the current schema version. Returns the migrated raw
// map and any error encountered.
func MigrateConfig(raw map[string]any, fromVersion int) (map[string]any, error) {
	needed, err := CheckMigration(fromVersion)
	if err != nil {
		return nil, err
	}
	if !needed {
		return raw, nil
	}

	current := fromVersion
	for _, m := range migrationChain {
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
// schema version and a migration path exists. A false result does not mean
// the version is current: unsupported (too new, zero or negative) versions
// also return false. Use CheckMigration to tell those cases apart.
func NeedsMigration(configVersion int) bool {
	needed, err := CheckMigration(configVersion)
	return err == nil && needed
}
