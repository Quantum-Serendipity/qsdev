package devinit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Display current .qsdev.yaml project configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigShow(cmd)
		},
	}
}

func runConfigShow(cmd *cobra.Command) error {
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}
	configPath := filepath.Join(projectRoot, ".qsdev.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(cmd.OutOrStdout(), "No .qsdev.yaml found. Run 'qsdev init' to create one.")
			return nil
		}
		return fmt.Errorf("reading .qsdev.yaml: %w", err)
	}
	_, err = cmd.OutOrStdout().Write(data)
	return err
}

func migrateCmd() *cobra.Command {
	var write bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate .qsdev.yaml to the latest schema version",
		Long: `Read .qsdev.yaml, apply any necessary schema migrations, and show the diff.

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
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}

	configPath := filepath.Join(projectRoot, ".qsdev.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading .qsdev.yaml: %w", err)
	}

	// Unmarshal to raw map for migration.
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing .qsdev.yaml: %w", err)
	}

	versionRaw, ok := raw["version"]
	if !ok {
		return fmt.Errorf("missing \"version\" field in .qsdev.yaml")
	}

	versionInt, ok := toConfigVersion(versionRaw)
	if !ok {
		return fmt.Errorf("field \"version\" must be an integer")
	}

	if !qsdevconfig.NeedsMigration(versionInt) {
		fmt.Fprintf(cmd.OutOrStdout(), ".qsdev.yaml is already at schema version %d (current). No migration needed.\n",
			types.ConfigVersionCurrent)
		return nil
	}

	migrated, err := qsdevconfig.MigrateConfig(raw, versionInt)
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
	fmt.Fprintf(cmd.OutOrStdout(), "--- .qsdev.yaml (before)\n+++ .qsdev.yaml (after)\n\n")
	fmt.Fprintln(cmd.OutOrStdout(), string(newData))

	if !write {
		fmt.Fprintln(cmd.OutOrStdout(), "Dry run. Use --write to apply the migration.")
		return nil
	}

	if err := fileutil.WriteFileAtomic(configPath, newData, 0o644); err != nil {
		return fmt.Errorf("writing migrated .qsdev.yaml: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Migrated .qsdev.yaml from version %d to %d.\n", versionInt, types.ConfigVersionCurrent)
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
