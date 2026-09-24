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

	"golang.org/x/sync/errgroup"

	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
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

// mcpProbeTimeout bounds each individual MCP server health probe so a single
// slow server cannot consume the doctor's overall budget.
const mcpProbeTimeout = 1500 * time.Millisecond

// mcpProbeConcurrency bounds how many MCP server probes run at once. Probing
// concurrently keeps checkMCP near max(probe) rather than sum(probe); the cap
// avoids spawning an unbounded number of subprocesses for large registries.
const mcpProbeConcurrency = 8

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
// availability, Nix installation, MCP server health, hook deployment, and
// permission consistency.
type doctorChecker struct {
	projectRoot string

	// timeout bounds the whole run; handle returns once it elapses even if a
	// check ignores its context. Overridable so tests need not wait 5s.
	timeout time.Duration

	// mcpServers lists the MCP server configs the doctor probes. Injectable so
	// tests can supply fakes without touching the project's .mcp.json.
	mcpServers func() ([]mcphealth.ServerConfig, error)
	// probeMCP performs one server health probe. Injectable for testing.
	probeMCP func(ctx context.Context, cfg mcphealth.ServerConfig) *mcphealth.ServerHealth
	// mcpCatalogErr reports why catalog-defined MCP servers are missing from
	// mcpServers, or nil. Injectable for testing.
	mcpCatalogErr func() error
}

func newDoctorChecker(projectRoot string) *doctorChecker {
	d := &doctorChecker{
		projectRoot: projectRoot,
		timeout:     doctorTimeout,
		probeMCP:    mcphealth.CheckServer,
		mcpCatalogErr: func() error {
			return mcpregistry.DefaultRegistry().CatalogErr()
		},
	}
	d.mcpServers = d.configuredMCPServers
	return d
}

// configuredMCPServers materializes the servers the project's .mcp.json
// configures as probe configs. Only the project's own servers are probed — not
// the whole built-in catalog — so the doctor reports on what this project
// actually runs (see mcpregistry.ConfiguredServers).
func (d *doctorChecker) configuredMCPServers() ([]mcphealth.ServerConfig, error) {
	return mcpregistry.ConfiguredServers(d.projectRoot, mcpregistry.DefaultRegistry())
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
// resolvable on PATH. It uses marker-file detection only (no container-runtime
// or environment probing), so it stays cheap and cannot stall on a hung daemon.
func (d *doctorChecker) checkTools(_ context.Context) checkResult {
	want := toolchainBinaries(ecosystem.DefaultRegistry().DetectAll(d.projectRoot).Project)
	if len(want) == 0 {
		return checkResult{"tools", checkPass, "no language toolchains required by detection", ""}
	}
	var missing []string
	for _, bin := range want {
		if _, err := exec.LookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	if len(missing) > 0 {
		return checkResult{"tools", checkFail, "missing on PATH: " + strings.Join(missing, ", "), "enter the devenv shell or install the toolchain"}
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

// checkMCP probes the health of each MCP server the project configures,
// concurrently and each within a bounded per-probe deadline, so several slow
// servers cannot serialize past the doctor's overall budget. Servers that are
// unsafe to start from a diagnostic (see mcpregistry.ProbeSkipReason) are listed as not
// probed rather than launched.
func (d *doctorChecker) checkMCP(ctx context.Context) checkResult {
	servers, err := d.mcpServers()
	if err != nil {
		return checkResult{"mcp", checkFail, err.Error(), "fix the JSON in .mcp.json"}
	}
	catalogErr := d.mcpCatalogErr()
	if len(servers) == 0 && catalogErr == nil {
		return checkResult{"mcp", checkPass, "no MCP servers configured", ""}
	}

	var probe []mcphealth.ServerConfig
	var skipped []string
	for _, cfg := range servers {
		if reason := mcpregistry.ProbeSkipReason(cfg); reason != "" {
			skipped = append(skipped, fmt.Sprintf("%s (%s)", cfg.Name, reason))
			continue
		}
		probe = append(probe, cfg)
	}

	healthyFlags := make([]bool, len(probe))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(mcpProbeConcurrency)
	for i, cfg := range probe {
		g.Go(func() error {
			probeCtx, cancel := context.WithTimeout(gctx, mcpProbeTimeout)
			defer cancel()
			healthyFlags[i] = d.probeMCP(probeCtx, cfg).Status == mcphealth.StatusHealthy
			return nil
		})
	}
	// Every probe returns nil, so Wait only surfaces context cancellation; the
	// flags are fully populated for all completed probes regardless.
	_ = g.Wait()

	healthy := 0
	for _, ok := range healthyFlags {
		if ok {
			healthy++
		}
	}
	unhealthy := len(probe) - healthy
	status := checkPass
	if unhealthy > 0 {
		status = checkWarn
	}
	detail := fmt.Sprintf("%d healthy, %d unhealthy of %d probed server(s)", healthy, unhealthy, len(probe))
	if len(skipped) > 0 {
		detail += fmt.Sprintf("; %d not probed: %s", len(skipped), strings.Join(skipped, ", "))
	}
	remediation := "investigate unhealthy servers with `qsdev mcp status`"
	if catalogErr != nil {
		// A broken catalog silently drops its server definitions (and the
		// required environment the probes rely on); never report that as a
		// clean pass.
		status = checkWarn
		detail += fmt.Sprintf("; catalog-defined servers are missing (%v)", catalogErr)
		remediation = "fix the MCP catalog or catalog override file named in the error"
	}
	return checkResult{"mcp", status, detail, remediation}
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
