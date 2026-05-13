package devenv

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/doctor"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/sysinfo"
)

func doctorCmd() *cobra.Command {
	var jsonOutput, checkMode bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check system prerequisites for development environment",
		Long: `Check that required and recommended tools are installed and meet
minimum version requirements. Outputs a formatted report of system info,
detected tools, and actionable recommendations.

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
	checks := doctor.RunAllChecks(ctx, osInfo)
	report := doctor.BuildReport(osInfo, checks, "0.1.0")

	w := cmd.OutOrStdout()

	if jsonOutput {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	if checkMode {
		var missing []string
		for _, t := range report.RequiredTools {
			if !t.Found || !t.VersionOK {
				missing = append(missing, t.Name)
			}
		}
		if len(missing) > 0 {
			_, _ = fmt.Fprintf(w, "Missing required tools: %s\n", strings.Join(missing, ", "))
			return fmt.Errorf("missing %d required tool(s)", len(missing))
		}
		_, _ = fmt.Fprintln(w, "All required tools are present.")
		return nil
	}

	doctor.FormatReport(w, report, doctor.UseColor(os.Stdout.Fd()))
	return nil
}
