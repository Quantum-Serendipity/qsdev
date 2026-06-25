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
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// doctorTimeout bounds the whole parallel diagnostic run.
const doctorTimeout = 5 * time.Second

// mcpProbeTimeout bounds each individual MCP server health probe so a single
// slow server cannot consume the doctor's overall budget.
const mcpProbeTimeout = 1500 * time.Millisecond

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
}

func newDoctorChecker(projectRoot string) *doctorChecker {
	return &doctorChecker{projectRoot: projectRoot}
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

// handle runs the selected checks in parallel (via errgroup) under a 5s timeout
// and returns the per-check results plus an aggregate verdict.
func (d *doctorChecker) handle(ctx context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	ctx, cancel := context.WithTimeout(ctx, doctorTimeout)
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

	results := make([]checkResult, len(selected))
	g, gctx := errgroup.WithContext(ctx)
	for i, c := range selected {
		g.Go(func() error {
			results[i] = c.run(gctx)
			return nil
		})
	}
	// Each check returns nil, so Wait only surfaces context cancellation; the
	// per-check results are always populated for completed checks.
	_ = g.Wait()

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
// resolvable on PATH.
func (d *doctorChecker) checkTools(_ context.Context) checkResult {
	det := detect.Detect(d.projectRoot)
	want := map[string]string{} // binary -> language
	if det.HasGoMod {
		want["go"] = "go"
	}
	if det.HasPackageJSON {
		want["node"] = "node"
	}
	if det.HasCargoToml {
		want["cargo"] = "rust"
	}
	if det.HasPyProject {
		want["python3"] = "python"
	}
	if len(want) == 0 {
		return checkResult{"tools", checkPass, "no language toolchains required by detection", ""}
	}
	var missing []string
	for bin := range want {
		if _, err := exec.LookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	if len(missing) > 0 {
		return checkResult{"tools", checkFail, "missing on PATH: " + strings.Join(missing, ", "), "enter the devenv shell or install the toolchain"}
	}
	return checkResult{"tools", checkPass, fmt.Sprintf("%d toolchain binary/binaries present", len(want)), ""}
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

// checkMCP probes each registered MCP server's health within a bounded deadline.
func (d *doctorChecker) checkMCP(ctx context.Context) checkResult {
	defs := mcpregistry.DefaultRegistry().All()
	if len(defs) == 0 {
		return checkResult{"mcp", checkPass, "no MCP servers registered", ""}
	}
	healthy, unhealthy := 0, 0
	for _, def := range defs {
		probeCtx, cancel := context.WithTimeout(ctx, mcpProbeTimeout)
		h := mcphealth.CheckServer(probeCtx, mcphealth.ServerConfig{
			Name: def.Name, Command: def.Command, Args: def.Args, URL: def.URL,
			Env: def.Env, RequiredEnv: def.RequiredEnv,
		})
		cancel()
		if h.Status == mcphealth.StatusHealthy {
			healthy++
		} else {
			unhealthy++
		}
	}
	status := checkPass
	if unhealthy > 0 {
		status = checkWarn
	}
	return checkResult{
		"mcp", status,
		fmt.Sprintf("%d healthy, %d unhealthy of %d registered server(s)", healthy, unhealthy, len(defs)),
		"investigate unhealthy servers with `qsdev mcp` diagnostics",
	}
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
