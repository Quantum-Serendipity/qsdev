// Package cloudisolation evaluates a project's cloud credential isolation
// (environment separation, credential file masking, agent deny rules) for
// each cloud provider the project configures. The evaluation is static: it
// reads the project's configuration files and never runs a cloud CLI, so
// `qsdev devenv doctor` and `qsdev check` (in CI) judge the same inputs.
package cloudisolation

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cloudcommon"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ClaudeSettingsRelPath is the project-relative, slash-separated path of the
// Claude Code settings file that holds the deny rules and sandbox read-deny
// list.
const ClaudeSettingsRelPath = ".claude/settings.json"

// Settings is the part of .claude/settings.json that carries credential
// isolation: permissions.deny and sandbox.filesystem.denyRead. Present is
// false when the project has no settings.json.
type Settings struct {
	Deny     []string
	DenyRead []string
	Present  bool
}

// ParseSettings reads the isolation keys of a settings.json document. Keys
// are matched exactly, as Claude Code reads them, so a decoy "Permissions"
// key cannot stand in for the real one.
func ParseSettings(data []byte) (Settings, error) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return Settings{}, fmt.Errorf("parsing %s: %w", ClaudeSettingsRelPath, err)
	}
	s := Settings{Present: true}
	perms, _ := root["permissions"].(map[string]any)
	s.Deny = stringList(perms["deny"])
	sandbox, _ := root["sandbox"].(map[string]any)
	fs, _ := sandbox["filesystem"].(map[string]any)
	s.DenyRead = stringList(fs["denyRead"])
	return s, nil
}

func stringList(v any) []string {
	items, _ := v.([]any)
	var out []string
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// ReadSettings reads the project's .claude/settings.json. A missing file
// yields empty settings with Present false (nothing is masked or denied); an
// unreadable or malformed one is an error.
func ReadSettings(projectRoot string) (Settings, error) {
	data, err := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(ClaudeSettingsRelPath)))
	if errors.Is(err, os.ErrNotExist) {
		return Settings{}, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("reading %s: %w", ClaudeSettingsRelPath, err)
	}
	return ParseSettings(data)
}

// ConfiguredProviders returns the cloud providers among the project's
// configured languages, in configuration order and without duplicates.
func ConfiguredProviders(languages []types.LanguageConfig) []cloudcommon.CloudProvider {
	var providers []cloudcommon.CloudProvider
	seen := make(map[cloudcommon.CloudProvider]bool)
	for _, lang := range languages {
		p, ok := cloudcommon.ProviderForModule(lang.Name)
		if !ok || seen[p] {
			continue
		}
		seen[p] = true
		providers = append(providers, p)
	}
	return providers
}

// Assess validates every isolation layer for each cloud provider the
// project configures. env holds the environment variables the project
// declares (its devenv modules); settings is the project's Claude Code
// settings; claudeCode says whether the project is set up for Claude Code
// and so expected to carry them. It returns nil when the project configures
// no cloud provider.
//
// The credential file masking and agent deny rules layers guard the Claude
// Code agent and live in .claude/settings.json. For a project without Claude
// Code and without that file (qsdev init --devenv-only) they do not apply, so
// only environment separation is reported. When Claude Code is expected, a
// missing file leaves those layers inactive.
func Assess(languages []types.LanguageConfig, env map[string]string, settings Settings, claudeCode bool) []cloudcommon.FailSafeReport {
	providers := ConfiguredProviders(languages)
	if len(providers) == 0 {
		return nil
	}
	reports := cloudcommon.ValidateAllProviders(providers, env, settings.Deny, settings.DenyRead)
	if settings.Present || claudeCode {
		return reports
	}
	for i := range reports {
		reports[i] = withoutAgentLayers(reports[i])
	}
	return reports
}

// withoutAgentLayers keeps only the layers of report that do not live in
// .claude/settings.json.
func withoutAgentLayers(report cloudcommon.FailSafeReport) cloudcommon.FailSafeReport {
	out := cloudcommon.FailSafeReport{Provider: report.Provider, AllLayersActive: true}
	for _, s := range report.Statuses {
		if Enforced(s.Layer) {
			continue
		}
		out.Statuses = append(out.Statuses, s)
		out.AllLayersActive = out.AllLayersActive && s.Active
	}
	return out
}

// Provider isolation statuses reported by Status.
const (
	StatusIsolated      = "isolated"
	StatusDegraded      = "degraded"
	StatusMisconfigured = "misconfigured"
)

// Enforced reports whether qsdev generates layer itself. The credential file
// masking and agent deny rules come from the generated .claude/settings.json,
// so a missing one means that configuration was weakened or is stale.
// Environment separation needs an account-specific value only the developer
// can supply, so its absence is advisory.
func Enforced(layer cloudcommon.FailSafeLayer) bool {
	return layer != cloudcommon.LayerEnvironmentSeparation
}

// Status summarises a provider report: StatusIsolated when every layer is
// active, StatusMisconfigured when an enforced layer is not, and
// StatusDegraded when only the advisory layer is missing.
func Status(report cloudcommon.FailSafeReport) string {
	status := StatusIsolated
	for _, s := range report.Statuses {
		if s.Active {
			continue
		}
		if Enforced(s.Layer) {
			return StatusMisconfigured
		}
		status = StatusDegraded
	}
	return status
}
