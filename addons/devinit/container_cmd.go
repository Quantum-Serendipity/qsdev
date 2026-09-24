package devinit

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/container"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

func containerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "container",
		Short: "Container runtime management tools",
		Long: `Tools for managing container runtimes and migrating from Docker to Podman.

Use "container detect" to show the active container runtime, and
"container migrate" to analyze compose files for Podman compatibility.`,
	}
	cmd.AddCommand(containerMigrateCmd(), containerDetectCmd())
	return cmd
}

func containerMigrateCmd() *cobra.Command {
	var (
		dryRun  bool
		autoFix bool
		asJSON  bool
		output  string
	)

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Analyze and fix Docker-to-Podman migration issues in compose files",
		Long: `Scans Docker Compose files for incompatibilities with Podman rootless mode.

By default runs in dry-run mode: shows issues without modifying files.
Use --auto-fix (or --dry-run=false) to apply all auto-fixable changes.
--auto-fix --dry-run lists the files the fixes would change without writing them.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			apply := migrateAppliesFixes(autoFix, dryRun, cmd.Flags().Changed("dry-run"))
			return runContainerMigrate(cmd.Context(), cmd, apply, autoFix && !apply, asJSON, output)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "Show issues without modifying files (--dry-run=false applies fixes)")
	cmd.Flags().BoolVar(&autoFix, "auto-fix", false, "Apply all auto-fixable changes (unless --dry-run is given explicitly)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output in JSON format")
	cmd.Flags().StringVar(&output, "output", "", "Write report to file instead of stdout")

	return cmd
}

// migrateAppliesFixes resolves whether `container migrate` rewrites compose
// files. An explicit --dry-run always wins: --dry-run=false applies fixes on
// its own and --auto-fix --dry-run only previews them. Otherwise --auto-fix
// applies fixes and the default is a dry run.
func migrateAppliesFixes(autoFix, dryRun, dryRunSet bool) bool {
	if dryRunSet {
		return !dryRun
	}
	return autoFix
}

func containerDetectCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect the active container runtime and capabilities",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runContainerDetect(cmd.Context(), cmd, asJSON)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "Output in JSON format")

	return cmd
}

func runContainerMigrate(ctx context.Context, cmd *cobra.Command, apply, previewFixes, asJSON bool, outputPath string) error {
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}

	prober := &container.ExecProber{}
	report, err := container.Analyze(ctx, projectRoot, prober)
	if err != nil {
		return fmt.Errorf("analyzing project: %w", err)
	}

	if len(report.ComposeFiles) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No compose files found in the project root.")
		return &ExitError{Code: 2}
	}

	switch {
	case apply:
		if err := applyComposeFixes(projectRoot, report); err != nil {
			return err
		}
		// Re-analyze after fixes to show updated report.
		report, err = container.Analyze(ctx, projectRoot, prober)
		if err != nil {
			return fmt.Errorf("re-analyzing after fixes: %w", err)
		}
	case previewFixes:
		if err := previewComposeFixes(cmd, report); err != nil {
			return err
		}
	}

	if err := writeMigrationReport(cmd, report, asJSON, outputPath); err != nil {
		return err
	}

	if report.Summary.Critical > 0 {
		return &ExitError{Code: 1}
	}
	return nil
}

// composeFixes returns the fixed content of file and whether it differs from
// what is on disk.
func composeFixes(file string, report *container.MigrationReport) (fixed []byte, changed bool, err error) {
	original, err := os.ReadFile(file)
	if err != nil {
		return nil, false, fmt.Errorf("reading %s: %w", file, err)
	}
	fixed, err = container.ApplyFixes(file, report.Issues)
	if err != nil {
		return nil, false, fmt.Errorf("applying fixes to %s: %w", file, err)
	}
	return fixed, !bytes.Equal(original, fixed), nil
}

// applyComposeFixes rewrites each compose file the fixes change, keeping its
// original permissions. Files the fixes leave unchanged are not touched. The
// compose files lie in projectRoot, and a rewrite through a symlink that
// resolves outside it is refused.
func applyComposeFixes(projectRoot string, report *container.MigrationReport) error {
	for _, file := range report.ComposeFiles {
		fixed, changed, err := composeFixes(file, report)
		if err != nil {
			return err
		}
		if !changed {
			continue
		}
		info, err := os.Stat(file)
		if err != nil {
			return fmt.Errorf("reading permissions of %s: %w", file, err)
		}
		rel, err := filepath.Rel(projectRoot, file)
		if err != nil {
			return fmt.Errorf("locating %s in %s: %w", file, projectRoot, err)
		}
		if err := fileutil.WriteFileAtomicInRoot(projectRoot, rel, fixed, info.Mode().Perm()); err != nil {
			return fmt.Errorf("writing fixed file %s: %w", file, err)
		}
	}
	return nil
}

// previewComposeFixes lists, on stderr so JSON output stays parseable, the
// compose files --auto-fix would rewrite, without writing them.
func previewComposeFixes(cmd *cobra.Command, report *container.MigrationReport) error {
	var changed []string
	for _, file := range report.ComposeFiles {
		_, differs, err := composeFixes(file, report)
		if err != nil {
			return err
		}
		if differs {
			changed = append(changed, file)
		}
	}

	w := cmd.ErrOrStderr()
	if len(changed) == 0 {
		fmt.Fprintln(w, "Dry run: --auto-fix would not change any compose file.")
		return nil
	}
	fmt.Fprintln(w, "Dry run: --auto-fix would modify these files (run without --dry-run to apply):")
	for _, file := range changed {
		fmt.Fprintf(w, "  %s\n", file)
	}
	return nil
}

// writeMigrationReport renders the report in full before writing it, so a
// rendering or write failure never leaves a truncated report file behind.
func writeMigrationReport(cmd *cobra.Command, report *container.MigrationReport, asJSON bool, outputPath string) error {
	format := container.FormatText
	if asJSON {
		format = container.FormatJSON
	}

	var buf bytes.Buffer
	useColor := !asJSON && outputPath == ""
	if err := container.FormatMigrationReport(report, format, &buf, useColor); err != nil {
		return fmt.Errorf("formatting report: %w", err)
	}

	if outputPath != "" {
		if err := fileutil.WriteFileAtomic(outputPath, buf.Bytes(), fileutil.ModeReadWrite); err != nil {
			return fmt.Errorf("writing report %s: %w", outputPath, err)
		}
		return nil
	}
	if _, err := cmd.OutOrStdout().Write(buf.Bytes()); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}
	return nil
}

func runContainerDetect(ctx context.Context, cmd *cobra.Command, asJSON bool) error {
	prober := &container.ExecProber{}
	info, err := container.Detect(ctx, prober)
	if err != nil {
		return fmt.Errorf("detecting container runtime: %w", err)
	}

	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}
	caps, err := container.DetectCapabilities(ctx, prober, info, projectRoot)
	if err != nil {
		return fmt.Errorf("detecting capabilities: %w", err)
	}

	w := cmd.OutOrStdout()

	if asJSON {
		report := &container.MigrationReport{
			RuntimeInfo:  info,
			Capabilities: caps,
		}
		return container.FormatMigrationReport(report, container.FormatJSON, w, false)
	}

	// Text output.
	fmt.Fprintf(w, "Active runtime: %s\n", info.Active)
	if info.Version != "" {
		fmt.Fprintf(w, "Version: %s\n", info.Version)
	}
	if info.Path != "" {
		fmt.Fprintf(w, "Path: %s\n", info.Path)
	}
	fmt.Fprintf(w, "Rootless: %v\n", info.Rootless)
	if info.SocketPath != "" {
		fmt.Fprintf(w, "Socket: %s\n", info.SocketPath)
	}
	fmt.Fprintf(w, "Compose method: %s\n", info.ComposeMethod)
	if info.HasDockerCompat {
		fmt.Fprintln(w, "Docker compatibility: active (docker is a Podman alias)")
	}

	if len(info.Available) > 1 {
		fmt.Fprintf(w, "Available runtimes: ")
		for i, r := range info.Available {
			if i > 0 {
				fmt.Fprint(w, ", ")
			}
			fmt.Fprint(w, r)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "\nCapabilities:")
	fmt.Fprintf(w, "  GPU passthrough: %v\n", caps.GPUPassthrough)
	fmt.Fprintf(w, "  NFS mounts in project: %v\n", caps.NFSMounts)
	fmt.Fprintf(w, "  Privileged ports: %v\n", caps.PrivilegedPorts)
	fmt.Fprintf(w, "  Rootless supported: %v\n", caps.RootlessSupported)
	fmt.Fprintf(w, "  User namespace configured: %v\n", caps.UserNamespaceConfigured)
	fmt.Fprintf(w, "  Cgroups v2: %v\n", caps.CgroupsV2)

	reasons := caps.NeedsRootfulFallback()
	if len(reasons) > 0 {
		fmt.Fprintln(w, "\nRootful fallback needed:")
		for _, r := range reasons {
			fmt.Fprintf(w, "  - %s\n", r)
		}
	}

	if len(info.Warnings) > 0 {
		fmt.Fprintln(w, "\nWarnings:")
		for _, warn := range info.Warnings {
			fmt.Fprintf(w, "  - %s\n", warn)
		}
	}

	return nil
}
