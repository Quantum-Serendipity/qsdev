package teamreport

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/cigeneration"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

const (
	// postureArtifactPrefix prefixes the per-project posture report artifact
	// name; the aggregator collects every artifact matching prefix + "*".
	postureArtifactPrefix = "posture-report-"

	// dashboardArtifact is the aggregator's own output artifact. The next run
	// restores the trend history from it.
	dashboardArtifact = "team-posture-dashboard"

	// historyFile is the trend history carried between aggregator runs.
	historyFile = "team-posture-history.json"

	// scopeFile lists the repositories to aggregate (see ScopeFile). It lives
	// in the repository that runs the aggregation workflow.
	scopeFile = "team-scope.json"

	// crossRepoTokenSecret names the secret holding a token that can read
	// Actions artifacts in, and open issues on, every repository in scope. The
	// workflow's own GITHUB_TOKEN is scoped to the aggregator repository and
	// can do neither.
	crossRepoTokenSecret = "TEAM_POSTURE_TOKEN"

	// unreleasedVersionPlaceholder is emitted when the generating binary is not
	// a release build, so the install step fails loudly until it is pinned.
	unreleasedVersionPlaceholder = "SET-A-RELEASED-VERSION"
)

// GenerateTeamWorkflow produces a complete GitHub Actions workflow YAML string
// for the team aggregation pipeline. The workflow collects the latest posture
// report artifact from every repository listed in the scope file, aggregates
// them, carries the trend history over from the previous run, and publishes
// the dashboard.
func GenerateTeamWorkflow() string {
	app := branding.Get().AppName
	var b strings.Builder

	b.WriteString("# Team security posture aggregation workflow\n")
	fmt.Fprintf(&b, "# %s team-report --generate-workflow\n", branding.GeneratedBy())
	b.WriteString("#\n")
	fmt.Fprintf(&b, "# Requires %s at the repository root, listing the repositories to\n", scopeFile)
	b.WriteString("# aggregate: {\"projects\": [{\"repo\": \"owner/name\"}]}\n")
	fmt.Fprintf(&b, "# Requires the %s secret: a token with actions:read on every listed\n", crossRepoTokenSecret)
	b.WriteString("# repository (and issues:write there to create issues). The default\n")
	b.WriteString("# GITHUB_TOKEN cannot read other repositories' artifacts.\n")
	b.WriteString("name: Team Posture Dashboard\n\n")

	writeTeamTriggers(&b)

	// Permissions for the workflow's own GITHUB_TOKEN: read this repository and
	// its previous runs' artifacts. Cross-repository access uses the scoped secret.
	b.WriteString("permissions:\n")
	b.WriteString("  contents: read\n")
	b.WriteString("  actions: read\n\n")

	b.WriteString("jobs:\n")
	b.WriteString("  aggregate:\n")
	b.WriteString("    name: Aggregate Posture Reports\n")
	b.WriteString("    runs-on: ubuntu-latest\n")
	b.WriteString("    steps:\n")

	writeHardenRunnerStep(&b)

	fmt.Fprintf(&b, "      - name: Checkout\n")
	fmt.Fprintf(&b, "        uses: %s %s\n", cigeneration.ActionCheckout, cigeneration.ActionCheckout.Comment())
	b.WriteString("\n")

	writeInstallStep(&b)
	writeCollectReportsStep(&b)
	writeRestoreHistoryStep(&b)

	b.WriteString("      - name: Aggregate posture reports\n")
	b.WriteString("        run: |\n")
	fmt.Fprintf(&b, "          %s team-report \\\n", app)
	b.WriteString("            --input-dir reports/ \\\n")
	b.WriteString("            --format md \\\n")
	b.WriteString("            --trend \\\n")
	fmt.Fprintf(&b, "            --history-file %s \\\n", historyFile)
	b.WriteString("            --output dashboard.md\n\n")

	b.WriteString("      - name: Generate JSON report\n")
	b.WriteString("        run: |\n")
	fmt.Fprintf(&b, "          %s team-report \\\n", app)
	b.WriteString("            --input-dir reports/ \\\n")
	b.WriteString("            --format json \\\n")
	b.WriteString("            --output team-posture.json\n\n")

	fmt.Fprintf(&b, "      - name: Upload dashboard\n")
	fmt.Fprintf(&b, "        uses: %s %s\n", cigeneration.ActionUploadArtifact, cigeneration.ActionUploadArtifact.Comment())
	b.WriteString("        with:\n")
	fmt.Fprintf(&b, "          name: %s\n", dashboardArtifact)
	b.WriteString("          path: |\n")
	b.WriteString("            dashboard.md\n")
	b.WriteString("            team-posture.json\n")
	fmt.Fprintf(&b, "            %s\n", historyFile)
	b.WriteString("          retention-days: 90\n\n")

	b.WriteString("      - name: Create issues for degraded projects\n")
	b.WriteString("        if: github.event.inputs.create-issues == 'true'\n")
	b.WriteString("        run: |\n")
	fmt.Fprintf(&b, "          %s team-report \\\n", app)
	b.WriteString("            --input-dir reports/ \\\n")
	fmt.Fprintf(&b, "            --history-file %s \\\n", historyFile)
	b.WriteString("            --create-issues\n")
	b.WriteString("        env:\n")
	fmt.Fprintf(&b, "          GH_TOKEN: ${{ secrets.%s }}\n", crossRepoTokenSecret)

	return b.String()
}

// writeTeamTriggers emits the aggregator's schedule and manual triggers.
func writeTeamTriggers(b *strings.Builder) {
	b.WriteString("on:\n")
	b.WriteString("  schedule:\n")
	b.WriteString("    - cron: '0 8 * * 1-5'  # Weekdays at 08:00 UTC\n")
	b.WriteString("  workflow_dispatch:\n")
	b.WriteString("    inputs:\n")
	b.WriteString("      create-issues:\n")
	b.WriteString("        description: 'Create GitHub issues for degraded projects'\n")
	b.WriteString("        type: boolean\n")
	b.WriteString("        default: false\n\n")
}

// writeHardenRunnerStep emits the pinned harden-runner step.
func writeHardenRunnerStep(b *strings.Builder) {
	fmt.Fprintf(b, "      - name: Harden Runner\n")
	fmt.Fprintf(b, "        uses: %s %s\n", cigeneration.ActionHardenRunner, cigeneration.ActionHardenRunner.Comment())
	b.WriteString("        with:\n")
	b.WriteString("          egress-policy: audit\n\n")
}

// writeInstallStep emits a step that installs a pinned release of the binary
// and verifies the archive against the release's checksums.txt before
// extracting it, instead of piping a remote script into a shell.
func writeInstallStep(b *strings.Builder) {
	cfg := branding.Get()
	app := cfg.AppName
	versionVar := strings.ToUpper(strings.ReplaceAll(app, "-", "_")) + "_VERSION"

	fmt.Fprintf(b, "      - name: Install %s\n", app)
	b.WriteString("        run: |\n")
	b.WriteString("          set -euo pipefail\n")
	fmt.Fprintf(b, "          asset=\"%s_${%s}_Linux_x86_64.tar.gz\"\n", app, versionVar)
	fmt.Fprintf(b, "          dir=\"$RUNNER_TEMP/%s-release\"\n", app)
	fmt.Fprintf(b, "          gh release download \"v${%s}\" --repo %s/%s \\\n", versionVar, cfg.GitHubOwner, cfg.GitHubRepo)
	b.WriteString("            --pattern \"$asset\" --pattern checksums.txt --dir \"$dir\"\n")
	b.WriteString("          (cd \"$dir\" && grep \"  ${asset}\\$\" checksums.txt | sha256sum --check --strict -)\n")
	fmt.Fprintf(b, "          tar -xzf \"$dir/$asset\" -C \"$dir\" %s\n", app)
	b.WriteString("          mkdir -p \"$HOME/.local/bin\"\n")
	fmt.Fprintf(b, "          install -m 0755 \"$dir/%s\" \"$HOME/.local/bin/%s\"\n", app, app)
	b.WriteString("          echo \"$HOME/.local/bin\" >> \"$GITHUB_PATH\"\n")
	b.WriteString("        env:\n")
	fmt.Fprintf(b, "          %s: \"%s\"\n", versionVar, pinnedReleaseVersion())
	b.WriteString("          GH_TOKEN: ${{ github.token }}\n\n")
}

// writeCollectReportsStep emits the step that downloads the latest posture
// report artifact from each repository in the scope file. download-artifact
// only sees the current run, so collection goes through `gh run download`
// with a token that can read the other repositories.
func writeCollectReportsStep(b *strings.Builder) {
	b.WriteString("      - name: Collect posture reports\n")
	b.WriteString("        run: |\n")
	b.WriteString("          set -euo pipefail\n")
	b.WriteString("          mkdir -p reports\n")
	fmt.Fprintf(b, "          for repo in $(jq -r '.projects[].repo' %s); do\n", scopeFile)
	fmt.Fprintf(b, "            if ! gh run download --repo \"$repo\" --pattern '%s*' --dir \"reports/${repo//\\//-}\"; then\n", postureArtifactPrefix)
	b.WriteString("              echo \"::warning::no posture report artifact found for $repo\"\n")
	b.WriteString("            fi\n")
	b.WriteString("          done\n")
	b.WriteString("        env:\n")
	fmt.Fprintf(b, "          GH_TOKEN: ${{ secrets.%s }}\n\n", crossRepoTokenSecret)
}

// writeRestoreHistoryStep emits the step that restores the trend history from
// the previous run's dashboard artifact, so score-drop alerts compare against
// real prior data instead of an empty history on every run.
func writeRestoreHistoryStep(b *strings.Builder) {
	b.WriteString("      - name: Restore posture history\n")
	b.WriteString("        run: |\n")
	b.WriteString("          set -euo pipefail\n")
	b.WriteString("          prev=\"$RUNNER_TEMP/previous-dashboard\"\n")
	fmt.Fprintf(b, "          if gh run download --repo \"$GITHUB_REPOSITORY\" --name %s --dir \"$prev\" \\\n", dashboardArtifact)
	fmt.Fprintf(b, "            && [ -f \"$prev/%s\" ]; then\n", historyFile)
	fmt.Fprintf(b, "            cp \"$prev/%s\" %s\n", historyFile, historyFile)
	b.WriteString("          else\n")
	b.WriteString("            echo \"No previous history found; starting a new one.\"\n")
	b.WriteString("          fi\n")
	b.WriteString("        env:\n")
	b.WriteString("          GH_TOKEN: ${{ github.token }}\n\n")
}

// pinnedReleaseVersion returns the release version the generated workflow
// installs: the version of the binary generating it, so the workflow runs the
// same release that produced it.
func pinnedReleaseVersion() string {
	v := strings.TrimPrefix(version.Info().Version, "v")
	if v == "" || v == "dev" || strings.Contains(v, "(devel)") {
		return unreleasedVersionPlaceholder
	}
	return v
}

// GeneratePerProjectSteps produces the GitHub Actions workflow YAML steps
// that each project should add to its CI pipeline to generate and upload
// a posture report artifact for consumption by the team aggregation workflow.
func GeneratePerProjectSteps() string {
	app := branding.Get().AppName
	var b strings.Builder

	b.WriteString("# Add these steps to each project's CI workflow.\n")
	b.WriteString("# They generate and upload a posture report for team aggregation.\n\n")

	writeHardenRunnerStep(&b)
	writeInstallStep(&b)

	// The report step records findings rather than gating on them: with the
	// default audit level a project with findings would exit non-zero, skip the
	// upload, and vanish from the dashboard. --scan populates vulnerability
	// counts, which are otherwise always zero.
	b.WriteString("      - name: Generate posture report\n")
	b.WriteString("        run: |\n")
	fmt.Fprintf(&b, "          %s status --scan --json --audit-level none > posture-report.json\n\n", app)

	fmt.Fprintf(&b, "      - name: Upload posture report\n")
	b.WriteString("        if: always()\n")
	fmt.Fprintf(&b, "        uses: %s %s\n", cigeneration.ActionUploadArtifact, cigeneration.ActionUploadArtifact.Comment())
	b.WriteString("        with:\n")
	fmt.Fprintf(&b, "          name: %s${{ github.repository_owner }}-${{ github.event.repository.name }}\n", postureArtifactPrefix)
	b.WriteString("          path: posture-report.json\n")
	b.WriteString("          retention-days: 30\n")

	return b.String()
}
