package devenv

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/container"
	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

func doctorCmd() *cobra.Command {
	var jsonOutput, checkMode bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check system prerequisites for development environment",
		Long: `Check that required and recommended tools are installed and meet
minimum version requirements. Outputs a formatted report of system info,
detected tools, and actionable recommendations. Inside a project it also
runs the health checks of the configured ecosystem modules statically: no
cloud CLI or other check command is executed.

Use --json for machine-readable output, or --check for a simple pass/fail
exit code (suitable for CI).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd, jsonOutput, checkMode)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output machine-readable JSON")
	cmd.Flags().BoolVar(&checkMode, "check", false, "Exit 0 if all required tools present, exit 1 if any missing")

	return cmd
}

func runDoctor(cmd *cobra.Command, jsonOutput, checkMode bool) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	osInfo := sysinfo.DetectOS()

	// An unknown working directory only disables the project-scoped checks
	// (NFS, cloud credential isolation, ecosystem module checks).
	projectRoot, _ := cmdutil.ProjectRoot()

	var containerSection *doctor.ContainerSection
	var sandboxSection *doctor.SandboxSection
	var wg sync.WaitGroup
	wg.Go(func() {
		containerSection = doctor.RunContainerCheck(ctx, &container.ExecProber{}, osInfo, projectRoot)
	})
	wg.Go(func() {
		sandboxSection = doctor.RunSandboxCheck(ctx, &sandbox.ExecSandboxProber{})
	})

	checks := doctor.RunAllChecks(ctx, osInfo)
	wg.Wait()

	report := doctor.BuildReport(osInfo, checks, version.Info().Version)
	report.SetContainerSection(containerSection)
	report.SetSandboxSection(sandboxSection)
	report.SetCloudSection(cloudIsolationSection(projectRoot))
	report.SetModuleCheckSection(moduleCheckSection(projectRoot, ecosystem.DefaultRegistry(), doctor.ModuleCheckEnv{
		LookupEnv: os.LookupEnv,
		LookPath:  exec.LookPath,
	}))
	slog.Info("doctor check complete",
		"required_tools", len(report.RequiredTools),
		"optional_tools", len(report.OptionalTools),
		"os", report.System.OS,
		"arch", report.System.Arch)

	w := cmd.OutOrStdout()
	return renderDoctorReport(w, report, jsonOutput, checkMode)
}

// renderDoctorReport writes report to w as JSON, a pass/fail check summary,
// or the formatted human report. In check mode it returns an error when any
// required tool is missing, including when the report is emitted as JSON.
func renderDoctorReport(w io.Writer, report *doctor.Report, jsonOutput, checkMode bool) error {
	missing := missingRequiredTools(report)

	if jsonOutput {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return fmt.Errorf("encoding doctor report: %w", err)
		}
		if checkMode && len(missing) > 0 {
			return fmt.Errorf("missing %d required tool(s): %s", len(missing), strings.Join(missing, ", "))
		}
		return nil
	}

	if checkMode {
		if len(missing) > 0 {
			_, _ = fmt.Fprintf(w, "Missing required tools: %s\n", strings.Join(missing, ", "))
			return fmt.Errorf("missing %d required tool(s)", len(missing))
		}
		_, _ = fmt.Fprintln(w, "All required tools are present.")
		return nil
	}

	doctor.FormatReport(w, report, writerUsesColor(w))
	return nil
}

// missingRequiredTools returns the names of required tools that are absent
// or below their minimum version.
func missingRequiredTools(report *doctor.Report) []string {
	var missing []string
	for _, t := range report.RequiredTools {
		if !t.Found || !t.VersionOK {
			missing = append(missing, t.Name)
		}
	}
	return missing
}

// writerUsesColor reports whether colored output suits w: only a terminal
// file gets color, never a buffer or pipe that the command was redirected to.
func writerUsesColor(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && doctor.UseColor(f.Fd())
}
