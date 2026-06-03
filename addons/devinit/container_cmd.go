package devinit

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/container"
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
Use --auto-fix to apply all auto-fixable changes (implies --dry-run=false).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runContainerMigrate(cmd.Context(), cmd, dryRun, autoFix, asJSON, output)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "Show issues without modifying files")
	cmd.Flags().BoolVar(&autoFix, "auto-fix", false, "Apply all auto-fixable changes")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output in JSON format")
	cmd.Flags().StringVar(&output, "output", "", "Write report to file instead of stdout")

	return cmd
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

func runContainerMigrate(ctx context.Context, cmd *cobra.Command, dryRun, autoFix, asJSON bool, outputPath string) error {
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

	// Determine output format and destination.
	format := container.FormatText
	if asJSON {
		format = container.FormatJSON
	}

	w := cmd.OutOrStdout()
	if outputPath != "" {
		f, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer f.Close()
		w = f
	}

	// If auto-fix is requested and dry-run is still the default, turn it off.
	if autoFix {
		dryRun = false
	}

	// Apply fixes if not dry-run.
	if !dryRun && autoFix {
		for _, file := range report.ComposeFiles {
			fixed, err := container.ApplyFixes(file, report.Issues)
			if err != nil {
				return fmt.Errorf("applying fixes to %s: %w", file, err)
			}
			if err := os.WriteFile(file, fixed, 0o644); err != nil {
				return fmt.Errorf("writing fixed file %s: %w", file, err)
			}
		}

		// Re-analyze after fixes to show updated report.
		report, err = container.Analyze(ctx, projectRoot, prober)
		if err != nil {
			return fmt.Errorf("re-analyzing after fixes: %w", err)
		}
	}

	useColor := !asJSON && outputPath == ""
	if err := container.FormatMigrationReport(report, format, w, useColor); err != nil {
		return fmt.Errorf("formatting report: %w", err)
	}

	if report.Summary.Critical > 0 {
		return &ExitError{Code: 1}
	}
	return nil
}

func runContainerDetect(ctx context.Context, cmd *cobra.Command, asJSON bool) error {
	prober := &container.ExecProber{}
	info, err := container.Detect(ctx, prober)
	if err != nil {
		return fmt.Errorf("detecting container runtime: %w", err)
	}

	caps, err := container.DetectCapabilities(ctx, prober, info)
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
	fmt.Fprintf(w, "  NFS mounts: %v\n", caps.NFSMounts)
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

	return nil
}
