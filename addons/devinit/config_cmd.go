package devinit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	gdevconfig "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/config"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/fileutil"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage .gdev.yaml project configuration",
		Long: `Commands for managing the .gdev.yaml project configuration file.

Use subcommands to migrate config schemas, validate configuration,
and inspect the current config state.`,
	}

	cmd.AddCommand(migrateCmd())

	return cmd
}

func migrateCmd() *cobra.Command {
	var write bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate .gdev.yaml to the latest schema version",
		Long: `Read .gdev.yaml, apply any necessary schema migrations, and show the diff.

By default, the command performs a dry run showing what would change.
Use --write to apply the migration in place.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMigrate(cmd, write)
		},
	}

	cmd.Flags().BoolVar(&write, "write", false, "Apply migration and write the updated file")

	return cmd
}

func runMigrate(cmd *cobra.Command, write bool) error {
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determining project root: %w", err)
	}

	configPath := filepath.Join(projectRoot, ".gdev.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading .gdev.yaml: %w", err)
	}

	// Unmarshal to raw map for migration.
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing .gdev.yaml: %w", err)
	}

	versionRaw, ok := raw["version"]
	if !ok {
		return fmt.Errorf("missing \"version\" field in .gdev.yaml")
	}

	versionInt, ok := toConfigVersion(versionRaw)
	if !ok {
		return fmt.Errorf("field \"version\" must be an integer")
	}

	if !gdevconfig.NeedsMigration(versionInt) {
		fmt.Fprintf(cmd.OutOrStdout(), ".gdev.yaml is already at schema version %d (current). No migration needed.\n",
			types.ConfigVersionCurrent)
		return nil
	}

	migrated, err := gdevconfig.MigrateConfig(raw, versionInt)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Marshal the migrated config.
	newData, err := yaml.Marshal(migrated)
	if err != nil {
		return fmt.Errorf("marshaling migrated config: %w", err)
	}

	// Show diff.
	fmt.Fprintf(cmd.OutOrStdout(), "Migration: version %d -> %d\n\n", versionInt, types.ConfigVersionCurrent)
	fmt.Fprintf(cmd.OutOrStdout(), "--- .gdev.yaml (before)\n+++ .gdev.yaml (after)\n\n")
	fmt.Fprintln(cmd.OutOrStdout(), string(newData))

	if !write {
		fmt.Fprintln(cmd.OutOrStdout(), "Dry run. Use --write to apply the migration.")
		return nil
	}

	if err := fileutil.WriteFileAtomic(configPath, newData, 0o644); err != nil {
		return fmt.Errorf("writing migrated .gdev.yaml: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Migrated .gdev.yaml from version %d to %d.\n", versionInt, types.ConfigVersionCurrent)
	return nil
}

// toConfigVersion converts a YAML-decoded value to int for version checks.
func toConfigVersion(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		if n == float64(int(n)) {
			return int(n), true
		}
		return 0, false
	default:
		return 0, false
	}
}
