package config

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/profile"
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

// migrationChain is the ordered list of migrations. It is unexported so
// importers cannot alter the migration path.
var migrationChain = []Migration{
	{
		FromVersion: 1,
		ToVersion:   2,
		Description: "split profile into infra_profile and project-type profile",
		Migrate:     migrateV1SplitProfile,
	},
}

// migrateV1SplitProfile moves an infrastructure profile name out of the v1
// `profile` key, which init filled with the infra profile, into v2's
// `infra_profile`. Only names of built-in infrastructure profiles move (that
// set is fixed, so the split is unambiguous); any other value was written by
// hand as the project-type profile the key now means, and stays.
func migrateV1SplitProfile(raw map[string]any) (map[string]any, error) {
	if _, ok := raw["infra_profile"]; ok {
		return nil, fmt.Errorf("infra_profile is not a version 1 key; set \"version: 2\" to use it")
	}
	name, ok := scalarString(raw["profile"])
	if !ok {
		return raw, nil
	}
	if _, isInfra := profile.DefaultProfileRegistry().Get(name); isInfra {
		raw["infra_profile"] = raw["profile"]
		delete(raw, "profile")
	}
	return raw, nil
}

// scalarString returns the string held by a raw config value: a decoded
// string, or a string scalar node when the document is migrated with its
// values kept as nodes (see migrateParsed).
func scalarString(v any) (string, bool) {
	switch v := v.(type) {
	case string:
		return v, true
	case *yaml.Node:
		if v.Kind == yaml.ScalarNode && v.ShortTag() == "!!str" {
			return v.Value, true
		}
	}
	return "", false
}

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
