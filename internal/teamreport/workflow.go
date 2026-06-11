package teamreport

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/cigeneration"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// SHA-pinned action reference for download-artifact (not in sha_pins.go yet).
var actionDownloadArtifact = cigeneration.ActionRef{
	Owner: "actions",
	Repo:  "download-artifact",
	SHA:   "95815c38cf2ff2164869cbab79da8d1f422bc89e",
	Tag:   "v4.2.1",
}

// GenerateTeamWorkflow produces a complete GitHub Actions workflow YAML string
// for the team aggregation pipeline. The workflow downloads posture reports
// from all projects, aggregates them, and publishes the dashboard.
func GenerateTeamWorkflow() string {
	var b strings.Builder

	b.WriteString("# Team security posture aggregation workflow\n")
	fmt.Fprintf(&b, "# %s team-report --generate-workflow\n", branding.GeneratedBy())
	b.WriteString("name: Team Posture Dashboard\n\n")

	// Triggers.
	b.WriteString("on:\n")
	b.WriteString("  schedule:\n")
	b.WriteString("    - cron: '0 8 * * 1-5'  # Weekdays at 08:00 UTC\n")
	b.WriteString("  workflow_dispatch:\n")
	b.WriteString("    inputs:\n")
	b.WriteString("      create-issues:\n")
	b.WriteString("        description: 'Create GitHub issues for degraded projects'\n")
	b.WriteString("        type: boolean\n")
	b.WriteString("        default: false\n\n")

	// Permissions.
	b.WriteString("permissions:\n")
	b.WriteString("  contents: read\n")
	b.WriteString("  issues: write\n")
	b.WriteString("  actions: read\n\n")

	// Jobs.
	b.WriteString("jobs:\n")
	b.WriteString("  aggregate:\n")
	b.WriteString("    name: Aggregate Posture Reports\n")
	b.WriteString("    runs-on: ubuntu-latest\n")
	b.WriteString("    steps:\n")

	// Step: Harden Runner.
	fmt.Fprintf(&b, "      - name: Harden Runner\n")
	fmt.Fprintf(&b, "        uses: %s %s\n", cigeneration.ActionHardenRunner, cigeneration.ActionHardenRunner.Comment())
	b.WriteString("        with:\n")
	b.WriteString("          egress-policy: audit\n\n")

	// Step: Checkout.
	fmt.Fprintf(&b, "      - name: Checkout\n")
	fmt.Fprintf(&b, "        uses: %s %s\n", cigeneration.ActionCheckout, cigeneration.ActionCheckout.Comment())
	b.WriteString("\n")

	// Step: Download posture reports.
	fmt.Fprintf(&b, "      - name: Download posture reports\n")
	fmt.Fprintf(&b, "        uses: %s %s\n", actionDownloadArtifact, actionDownloadArtifact.Comment())
	b.WriteString("        with:\n")
	b.WriteString("          pattern: posture-report-*\n")
	b.WriteString("          path: reports/\n")
	b.WriteString("          merge-multiple: true\n\n")

	// Step: Install qsdev.
	b.WriteString("      - name: Install qsdev\n")
	b.WriteString("        run: |\n")
	fmt.Fprintf(&b, "          curl -sSfL %s | sh\n", branding.InstallScriptURL())
	b.WriteString("          echo \"$HOME/.local/bin\" >> $GITHUB_PATH\n\n")

	// Step: Aggregate reports.
	b.WriteString("      - name: Aggregate posture reports\n")
	b.WriteString("        run: |\n")
	b.WriteString("          qsdev team-report \\\n")
	b.WriteString("            --input-dir reports/ \\\n")
	b.WriteString("            --format md \\\n")
	b.WriteString("            --trend \\\n")
	b.WriteString("            --history-file team-posture-history.json \\\n")
	b.WriteString("            --output dashboard.md\n\n")

	// Step: Generate JSON report.
	b.WriteString("      - name: Generate JSON report\n")
	b.WriteString("        run: |\n")
	b.WriteString("          qsdev team-report \\\n")
	b.WriteString("            --input-dir reports/ \\\n")
	b.WriteString("            --format json \\\n")
	b.WriteString("            --output team-posture.json\n\n")

	// Step: Upload dashboard artifact.
	fmt.Fprintf(&b, "      - name: Upload dashboard\n")
	fmt.Fprintf(&b, "        uses: %s %s\n", cigeneration.ActionUploadArtifact, cigeneration.ActionUploadArtifact.Comment())
	b.WriteString("        with:\n")
	b.WriteString("          name: team-posture-dashboard\n")
	b.WriteString("          path: |\n")
	b.WriteString("            dashboard.md\n")
	b.WriteString("            team-posture.json\n")
	b.WriteString("            team-posture-history.json\n\n")

	// Step: Create issues (conditional).
	b.WriteString("      - name: Create issues for degraded projects\n")
	b.WriteString("        if: github.event.inputs.create-issues == 'true'\n")
	b.WriteString("        run: |\n")
	b.WriteString("          qsdev team-report \\\n")
	b.WriteString("            --input-dir reports/ \\\n")
	b.WriteString("            --create-issues\n")
	b.WriteString("        env:\n")
	b.WriteString("          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}\n")

	return b.String()
}

// GeneratePerProjectSteps produces the GitHub Actions workflow YAML steps
// that each project should add to its CI pipeline to generate and upload
// a posture report artifact for consumption by the team aggregation workflow.
func GeneratePerProjectSteps() string {
	var b strings.Builder

	b.WriteString("# Add these steps to each project's CI workflow.\n")
	b.WriteString("# They generate and upload a posture report for team aggregation.\n\n")

	// Step: Harden Runner.
	fmt.Fprintf(&b, "      - name: Harden Runner\n")
	fmt.Fprintf(&b, "        uses: %s %s\n", cigeneration.ActionHardenRunner, cigeneration.ActionHardenRunner.Comment())
	b.WriteString("        with:\n")
	b.WriteString("          egress-policy: audit\n\n")

	// Step: Generate posture report.
	b.WriteString("      - name: Generate posture report\n")
	b.WriteString("        run: |\n")
	b.WriteString("          qsdev status --json > posture-report.json\n\n")

	// Step: Upload posture report.
	fmt.Fprintf(&b, "      - name: Upload posture report\n")
	fmt.Fprintf(&b, "        uses: %s %s\n", cigeneration.ActionUploadArtifact, cigeneration.ActionUploadArtifact.Comment())
	b.WriteString("        with:\n")
	b.WriteString("          name: posture-report-${{ github.repository_owner }}-${{ github.event.repository.name }}\n")
	b.WriteString("          path: posture-report.json\n")
	b.WriteString("          retention-days: 30\n")

	return b.String()
}
