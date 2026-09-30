package status

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules" // registers the modules checkTools detects with
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// doctorTimeout bounds the whole parallel diagnostic run.
const doctorTimeout = 5 * time.Second

const (
	checkPass = "pass"
	checkWarn = "warning"
	checkFail = "fail"
)

// checkResult is the outcome of one diagnostic check.
type checkResult struct {
	Name        string `json:"name"`
	Status      string `json:"status"` // pass | warning | fail
	Detail      string `json:"detail"`
	Remediation string `json:"remediation,omitempty"`
}

// doctorChecker runs the integrity diagnostics for the project (distinct from the
// system-prerequisite doctor): config validity, state integrity, tool
// availability, Nix installation, MCP config validity, hook deployment, and
// permission consistency.
type doctorChecker struct {
	projectRoot string

	// timeout bounds the whole run; handle returns once it elapses even if a
	// check ignores its context. Overridable so tests need not wait 5s.
	timeout time.Duration
}

func newDoctorChecker(projectRoot string) *doctorChecker {
	return &doctorChecker{projectRoot: projectRoot, timeout: doctorTimeout}
}

// namedCheck pairs a check's stable name with its implementation.
type namedCheck struct {
	name string
	run  func(ctx context.Context) checkResult
}

// checks returns the seven diagnostic checks in stable order.
func (d *doctorChecker) checks() []namedCheck {
	return []namedCheck{
		{"config", d.checkConfig},
		{"state", d.checkState},
		{"tools", d.checkTools},
		{"nix", d.checkNix},
		{"mcp", d.checkMCP},
		{"hooks", d.checkHooks},
		{"permissions", d.checkPermissions},
	}
}

// handle runs the selected checks in parallel under the doctor's timeout and
// returns the per-check results plus an aggregate verdict.
func (d *doctorChecker) handle(ctx context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	all := d.checks()
	selected := all
	if only := toolutil.StringArgOr(req.Arguments, "check", ""); only != "" {
		selected = nil
		for _, c := range all {
			if c.name == only {
				selected = append(selected, c)
			}
		}
		if len(selected) == 0 {
			return toolutil.NotConfigured("unknown check",
				map[string]any{"got": only, "allowed": checkNames(all)}), nil
		}
	}

	results := d.runChecks(ctx, selected)

	pass, warn, fail := 0, 0, 0
	for _, r := range results {
		switch r.Status {
		case checkPass:
			pass++
		case checkWarn:
			warn++
		case checkFail:
			fail++
		}
	}
	overall := checkPass
	if fail > 0 {
		overall = checkFail
	} else if warn > 0 {
		overall = checkWarn
	}

	structured := map[string]any{
		"overall": overall,
		"checks":  results,
		"summary": map[string]int{"pass": pass, "warning": warn, "fail": fail},
	}
	text := fmt.Sprintf("devenv_doctor: %s — %d pass, %d warning, %d fail", overall, pass, warn, fail)
	return toolutil.Result(text, structured), nil
}

// runChecks runs every check concurrently and returns their results in input
// order. It returns as soon as ctx is done even when a check ignores ctx (e.g. a
// hung subprocess), filling each unfinished check with a timed-out failure so the
// advertised deadline actually bounds the tool call. A check that finishes late
// writes into the buffered channel and is discarded.
func (d *doctorChecker) runChecks(ctx context.Context, checks []namedCheck) []checkResult {
	type indexed struct {
		i int
		r checkResult
	}
	done := make(chan indexed, len(checks))
	for i, c := range checks {
		go func() { done <- indexed{i, c.run(ctx)} }()
	}

	results := make([]checkResult, len(checks))
	finished := make([]bool, len(checks))
	for remaining := len(checks); remaining > 0; remaining-- {
		select {
		case o := <-done:
			results[o.i], finished[o.i] = o.r, true
		case <-ctx.Done():
			for i, c := range checks {
				if !finished[i] {
					results[i] = checkResult{c.name, checkFail,
						fmt.Sprintf("timed out after %s", d.timeout),
						"re-run the check alone with `check: " + c.name + "` to investigate"}
				}
			}
			return results
		}
	}
	return results
}

// checkConfig parses the project config and the optional local overlay.
func (d *doctorChecker) checkConfig(_ context.Context) checkResult {
	cfgPath := filepath.Join(d.projectRoot, branding.Get().ConfigFile)
	if _, err := os.Stat(cfgPath); err != nil {
		return checkResult{"config", checkWarn, branding.Get().ConfigFile + " not found", "run `qsdev init` to create it"}
	}
	if _, err := config.ParseQsdevConfig(cfgPath); err != nil {
		return checkResult{"config", checkFail, "parse error: " + err.Error(), "fix the YAML in " + branding.Get().ConfigFile}
	}
	localPath := filepath.Join(d.projectRoot, branding.Get().LocalConfig)
	if _, err := os.Stat(localPath); err == nil {
		if _, err := config.ParseLocalConfig(localPath); err != nil {
			return checkResult{"config", checkFail, branding.Get().LocalConfig + " parse error: " + err.Error(), "fix the YAML in " + branding.Get().LocalConfig}
		}
	}
	return checkResult{"config", checkPass, branding.Get().ConfigFile + " is valid", ""}
}

// checkState verifies the generated-file state ledger is present and parseable.
func (d *doctorChecker) checkState(_ context.Context) checkResult {
	statePath := filepath.Join(d.projectRoot, state.StateFilePaths()[0])
	if _, err := os.Stat(statePath); err != nil {
		return checkResult{"state", checkWarn, "state file absent (project not initialized)", "run `qsdev init`"}
	}
	st, err := state.LoadStateFromFile(statePath)
	if err != nil {
		return checkResult{"state", checkFail, "state unreadable: " + err.Error(), "re-run `qsdev init --update`"}
	}
	return checkResult{"state", checkPass, fmt.Sprintf("state ok (%d files tracked)", len(st.Files)), ""}
}

// checkTools verifies the primary toolchain binary for each detected language is
// resolvable on PATH, and that each toolchain matches what the project's files
// require (see ecosystem.ToolchainChecker). It uses marker-file detection only
// (no container-runtime or environment probing), so it stays cheap and cannot
// stall on a hung daemon.
func (d *doctorChecker) checkTools(ctx context.Context) checkResult {
	reg := ecosystem.DefaultRegistry()
	summary := reg.DetectAll(d.projectRoot)
	want := toolchainBinaries(summary.Project)
	var missing []string
	for _, bin := range want {
		if _, err := exec.LookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	if len(missing) > 0 {
		return checkResult{"tools", checkFail, "missing on PATH: " + strings.Join(missing, ", "), "enter the devenv shell or install the toolchain"}
	}
	if mismatched := reg.ToolchainWarnings(ctx, d.projectRoot, summary.Results); len(mismatched) > 0 {
		return checkResult{"tools", checkWarn, "toolchain mismatch: " + strings.Join(mismatched, "; "), "follow each warning's remedy, then re-enter the devenv shell"}
	}
	if len(want) == 0 {
		return checkResult{"tools", checkPass, "no language toolchains required by detection", ""}
	}
	return checkResult{"tools", checkPass, fmt.Sprintf("%d toolchain binary/binaries present", len(want)), ""}
}

// toolchainBinaries returns the toolchain binaries the detected languages need
// on PATH, covering the same languages as toolutil.DetectedLanguages. Maven and
// Gradle projects usually ship a wrapper (mvnw/gradlew), so the JVM is what both
// require.
func toolchainBinaries(det types.DetectedProject) []string {
	var bins []string
	add := func(present bool, bin string) {
		if present {
			bins = append(bins, bin)
		}
	}
	add(det.HasGoMod, "go")
	add(det.HasPackageJSON, "node")
	add(det.HasCargoToml, "cargo")
	add(det.HasPyProject, "python3")
	add(det.HasPomXML || det.HasBuildGradle, "java")
	add(det.HasCsproj, "dotnet")
	return bins
}

// checkNix verifies the nix binary and store are available.
func (d *doctorChecker) checkNix(_ context.Context) checkResult {
	if _, err := exec.LookPath("nix"); err != nil {
		return checkResult{"nix", checkWarn, "nix not on PATH", "install Nix to use nix_run and devenv"}
	}
	if info, err := os.Stat("/nix/store"); err != nil || !info.IsDir() {
		return checkResult{"nix", checkWarn, "nix binary present but /nix/store is missing", "verify the Nix installation"}
	}
	return checkResult{"nix", checkPass, "nix installed and store accessible", ""}
}

// mcpLivenessHint ends every MCP check result: the check is static, so
// whether a server actually starts is left to `qsdev mcp status`.
const mcpLivenessHint = "liveness: run `qsdev mcp status` (starts trusted definitions only)"

// checkMCP validates the project's .mcp.json statically (see
// doctor.MCPFindings). It starts no server and dials no URL: .mcp.json is
// repository content and may name any command or endpoint.
func (d *doctorChecker) checkMCP(_ context.Context) checkResult {
	return summarizeMCP(doctor.MCPFindings(d.projectRoot, mcpregistry.DefaultRegistry()))
}

// summarizeMCP maps the static MCP findings onto the check's result. Any
// misconfigured or degraded server, or a section warning such as a catalog
// that failed to load, makes it a warning rather than a clean pass.
func summarizeMCP(ms *doctor.MCPSection, err error) checkResult {
	if err != nil {
		return checkResult{"mcp", checkFail, err.Error(), "fix the JSON in .mcp.json; " + mcpLivenessHint}
	}
	if ms == nil || len(ms.Servers) == 0 {
		return checkResult{"mcp", checkPass, "no MCP servers configured", mcpLivenessHint}
	}

	misconfigured, degraded := 0, 0
	var problems []string
	for _, srv := range ms.Servers {
		switch srv.Status {
		case doctor.MCPStatusMisconfigured:
			misconfigured++
		case doctor.MCPStatusDegraded:
			degraded++
		default:
			continue
		}
		problem := srv.DisplayName()
		if len(srv.Issues) > 0 {
			problem += " (" + srv.Issues[0].Message + ")"
		}
		problems = append(problems, problem)
	}

	detail := fmt.Sprintf("%d configured; %d misconfigured, %d degraded", len(ms.Servers), misconfigured, degraded)
	if len(problems) > 0 {
		detail += ": " + strings.Join(problems, ", ")
	}
	if len(ms.Warnings) > 0 {
		detail += "; " + strings.Join(ms.Warnings, "; ")
	}
	status := checkPass
	if len(problems) > 0 || len(ms.Warnings) > 0 {
		status = checkWarn
	}
	return checkResult{"mcp", status, detail, mcpLivenessHint}
}

// checkHooks verifies hook files are deployed under .claude/hooks/.
func (d *doctorChecker) checkHooks(_ context.Context) checkResult {
	hooksDir := filepath.Join(d.projectRoot, ".claude", "hooks")
	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		return checkResult{"hooks", checkWarn, ".claude/hooks not present", "run `qsdev init` to deploy security hooks"}
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			count++
		}
	}
	if count == 0 {
		return checkResult{"hooks", checkWarn, ".claude/hooks is empty", "run `qsdev init --update` to deploy hooks"}
	}
	return checkResult{"hooks", checkPass, fmt.Sprintf("%d hook file(s) deployed", count), ""}
}

// checkPermissions verifies the Claude settings carry deny rules.
func (d *doctorChecker) checkPermissions(_ context.Context) checkResult {
	settingsPath := filepath.Join(d.projectRoot, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath) //nolint:gosec // project-root settings file
	if err != nil {
		return checkResult{"permissions", checkWarn, ".claude/settings.json not present", "run `qsdev init`"}
	}
	var settings struct {
		Permissions struct {
			Deny  []string `json:"deny"`
			Allow []string `json:"allow"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return checkResult{"permissions", checkFail, "settings.json is invalid JSON: " + err.Error(), "fix the JSON syntax"}
	}
	if len(settings.Permissions.Deny) == 0 {
		return checkResult{"permissions", checkWarn, "no deny rules configured in settings.json", "add security deny rules via qsdev"}
	}
	return checkResult{"permissions", checkPass, fmt.Sprintf("%d deny rule(s) configured", len(settings.Permissions.Deny)), ""}
}

// checkNames lists the names of the provided checks for error reporting.
func checkNames(checks []namedCheck) []string {
	out := make([]string, len(checks))
	for i, c := range checks {
		out[i] = c.name
	}
	return out
}
