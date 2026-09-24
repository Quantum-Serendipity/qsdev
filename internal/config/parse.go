// Package config provides parsing, validation, and migration for .qsdev.yaml
// project configuration files.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/profile"
	"github.com/Quantum-Serendipity/qsdev/internal/validation"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
	"gopkg.in/yaml.v3"
)

// ValidationError describes a single validation failure in a QsdevConfig.
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("%s: %q — %s", e.Field, e.Value, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateOptions provides additional context for config validation.
type ValidateOptions struct {
	// ProfileNames are the project-type profiles `profile` must name (the
	// devinit project-profile registry, which embedders can extend).
	ProfileNames []string
	// ToolNames are the qsdev catalog tools tools.enabled and tools.disabled
	// must name.
	ToolNames []string
	// MCPToolNames are the tools qsdev's MCP server can mount
	// (mcpserve.MountableToolNames); mcp.disabled_tools must name one of them.
	// Validation of that list is skipped when none are given.
	MCPToolNames []string
}

// ParseQsdevConfig reads and parses a .qsdev.yaml file at path.
//
// It uses two-pass parsing: first unmarshal to map[string]any to extract
// and validate the version field with clear error messages, then a strict
// (known-field) struct unmarshal into QsdevConfig. Unknown/misspelled YAML
// keys are rejected with an error rather than silently dropped, so a typo in
// a security key (e.g. "script_blockng:") cannot silently discard its setting.
// A config at an older supported schema version is migrated to the current
// schema through the migration chain before it is returned.
func ParseQsdevConfig(path string) (*types.QsdevConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	cfg, err := ParseQsdevConfigBytes(data)
	if err != nil {
		slog.Debug("config parse failed", "path", path, "error", err)
		return nil, err
	}
	slog.Debug("config loaded", "path", path, "version", cfg.Version)
	return cfg, nil
}

// ParseQsdevConfigBytes parses .qsdev.yaml content from raw bytes.
func ParseQsdevConfigBytes(data []byte) (*types.QsdevConfig, error) {
	// Pass 1: extract version field from raw map.
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	versionRaw, ok := raw["version"]
	if !ok {
		return nil, fmt.Errorf("missing required field \"version\" in %s; add \"version: %d\" at the top of the file",
			branding.Get().ConfigFile, types.ConfigVersionCurrent)
	}

	versionInt, ok := toInt(versionRaw)
	if !ok {
		return nil, fmt.Errorf("field \"version\" must be an integer, got %T", versionRaw)
	}

	if versionInt > types.ConfigVersionMax {
		return nil, fmt.Errorf(
			"config version %d is newer than this binary supports (max %d); please update %s to the latest version",
			versionInt, types.ConfigVersionMax, branding.Get().AppName)
	}

	if versionInt < types.ConfigVersionMin {
		return nil, fmt.Errorf(
			"config version %d is no longer supported (minimum %d); run \"%s config migrate\" to upgrade",
			versionInt, types.ConfigVersionMin, branding.Get().AppName)
	}

	// Pass 2: strict (known-field) struct unmarshal. KnownFields(true) rejects
	// any unknown/misspelled key rather than silently dropping it, so a typo'd
	// top-level or nested security key surfaces as an error instead of quietly
	// discarding the user's intended setting (config/security data loss).
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var cfg types.QsdevConfig
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Reject any YAML document after the first: both passes above read only
	// the first document, so a second one (e.g. a pasted snippet after
	// "---") would otherwise be silently ignored, security settings included.
	more, err := hasMoreDocuments(dec)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if more {
		return nil, fmt.Errorf("parsing config: multiple YAML documents are not supported; "+
			"merge everything after the first \"---\" into a single document in %s", branding.Get().ConfigFile)
	}

	if versionInt < types.ConfigVersionCurrent {
		return migrateParsed(data, versionInt)
	}
	return &cfg, nil
}

// migrateParsed brings an older-schema config to the current schema in
// memory, so callers only ever see the current layout (cfg.Version is the
// current version). The file itself is rewritten by the next config write
// (`qsdev init --update` or `qsdev config migrate --write`). The document was
// already strictly decoded, so any error here is a migration failure rather
// than a user typo.
//
// The migration steps see each top-level value as the document's own node,
// so re-encoding reproduces every scalar exactly as written; a round trip
// through plain Go values would turn an unquoted `version: 3.10` into 3.1.
func migrateParsed(data []byte, fromVersion int) (*types.QsdevConfig, error) {
	fail := func(err error) (*types.QsdevConfig, error) {
		return nil, fmt.Errorf("migrating %s: %w", branding.Get().ConfigFile, err)
	}
	var nodes map[string]yaml.Node
	if err := yaml.Unmarshal(data, &nodes); err != nil {
		return fail(err)
	}
	raw := make(map[string]any, len(nodes))
	for k, n := range nodes {
		raw[k] = &n
	}
	migrated, err := MigrateConfig(raw, fromVersion)
	if err != nil {
		return fail(err)
	}
	out, err := yaml.Marshal(migrated)
	if err != nil {
		return fail(err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(out))
	dec.KnownFields(true)
	var cfg types.QsdevConfig
	if err := dec.Decode(&cfg); err != nil {
		return fail(err)
	}
	return &cfg, nil
}

// hasMoreDocuments reports whether dec holds another YAML document with
// content. Empty trailing documents (a closing "---", optionally followed by
// comments) carry nothing that could be ignored, so they are skipped.
func hasMoreDocuments(dec *yaml.Decoder) (bool, error) {
	for {
		var doc yaml.Node
		if err := dec.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				return false, nil
			}
			return false, err
		}
		if len(doc.Content) > 0 && doc.Content[0].Tag != "!!null" {
			return true, nil
		}
	}
}

// ValidateQsdevConfig validates a parsed config and returns all validation
// errors found. An empty slice means the config is valid.
func ValidateQsdevConfig(cfg *types.QsdevConfig, opts ValidateOptions) []ValidationError {
	var errs []ValidationError

	// Validate language names.
	for i, lang := range cfg.Languages {
		if lang.Name == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("languages[%d].name", i),
				Message: "language name is required",
			})
		} else if !validation.IsValidLanguage(lang.Name) {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("languages[%d].name", i),
				Value:   lang.Name,
				Message: "unknown language; see supported languages list",
			})
		}
	}

	// Validate service names.
	for i, svc := range cfg.Services {
		if svc.Name == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("services[%d].name", i),
				Message: "service name is required",
			})
		} else if !validation.IsValidService(svc.Name) {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("services[%d].name", i),
				Value:   svc.Name,
				Message: "unknown service; see supported services list",
			})
		}
	}

	errs = append(errs, validateSplicedValues(cfg)...)

	// Validate security level.
	if cfg.Security.Level != "" && !validation.IsValidSecurityLevel(cfg.Security.Level) {
		errs = append(errs, ValidationError{
			Field:   "security.level",
			Value:   cfg.Security.Level,
			Message: "invalid security level; " + validValues(validation.SecurityLevels()),
		})
	}

	// Validate tier: an unknown explicit tier must fail validation rather
	// than be silently replaced by an inferred one during generation.
	if cfg.Tier != "" && !validation.IsValidTier(cfg.Tier) {
		errs = append(errs, ValidationError{
			Field:   "tier",
			Value:   cfg.Tier,
			Message: "unknown tier; " + validValues(validation.Tiers()),
		})
	}

	// Validate claude_code.permission_level.
	if cfg.ClaudeCode.PermissionLevel != "" && !validation.IsValidPermissionPreset(cfg.ClaudeCode.PermissionLevel) {
		errs = append(errs, ValidationError{
			Field:   "claude_code.permission_level",
			Value:   cfg.ClaudeCode.PermissionLevel,
			Message: "invalid permission level; " + validValues(validation.PermissionPresets()),
		})
	}

	// Validate tools.enabled against known tool names.
	if len(opts.ToolNames) > 0 {
		knownTools := toSet(opts.ToolNames)
		for _, t := range cfg.Tools.Enabled {
			if !knownTools[t] {
				errs = append(errs, ValidationError{
					Field:   "tools.enabled",
					Value:   t,
					Message: "unknown tool name",
				})
			}
		}
		mcpTools := toSet(opts.MCPToolNames)
		for _, t := range cfg.Tools.Disabled {
			if !knownTools[t] {
				msg := "unknown tool name"
				if mcpTools[t] {
					msg = "is an MCP tool name, not a catalog tool; list it under mcp.disabled_tools to deny it"
				}
				errs = append(errs, ValidationError{
					Field:   "tools.disabled",
					Value:   t,
					Message: msg,
				})
			}
		}
	}

	errs = append(errs, validateMCPDisabledTools(cfg, opts)...)
	errs = append(errs, validateCredentialVend(cfg.Security.CredentialVend)...)
	errs = append(errs, validateProfiles(cfg, opts)...)
	errs = append(errs, validateHooks(cfg.Hooks)...)

	// git.branch_pattern is spliced into the branch-naming pre-push hook.
	if err := validation.CheckBranchPattern(cfg.Git.BranchPattern); err != nil {
		errs = append(errs, ValidationError{
			Field:   "git.branch_pattern",
			Value:   cfg.Git.BranchPattern,
			Message: err.Error(),
		})
	}

	// Validate qsdev_version syntax (if present).
	if cfg.QsdevVersion != "" {
		if _, err := ParseVersionConstraint(cfg.QsdevVersion); err != nil {
			errs = append(errs, ValidationError{
				Field:   "qsdev_version",
				Value:   cfg.QsdevVersion,
				Message: fmt.Sprintf("invalid version constraint: %v", err),
			})
		}
	}

	// Validate client fields.
	if cfg.Client != nil {
		if cfg.Client.Name == "" {
			errs = append(errs, ValidationError{
				Field:   "client.name",
				Message: "client name is required when client block is present",
			})
		}
		if cfg.Client.SecurityLevel != "" && !validation.IsValidSecurityLevel(cfg.Client.SecurityLevel) {
			errs = append(errs, ValidationError{
				Field:   "client.security_level",
				Value:   cfg.Client.SecurityLevel,
				Message: "invalid security level; " + validValues(validation.SecurityLevels()),
			})
		}
		if cfg.Client.DataClassification != "" && !validation.IsValidDataClassification(cfg.Client.DataClassification) {
			errs = append(errs, ValidationError{
				Field:   "client.data_classification",
				Value:   cfg.Client.DataClassification,
				Message: "invalid data classification; " + validValues(validation.DataClassifications()),
			})
		}
	}

	return errs
}

// validateMCPDisabledTools checks mcp.disabled_tools against the MCP tool
// namespace (opts.MCPToolNames). An unknown name is an error rather than a
// harmless no-op: the deny it was meant to install (typically a misspelled
// tool) would otherwise leave the intended tool runnable. Catalog tool names
// belong in tools.disabled and are rejected here the same way.
func validateMCPDisabledTools(cfg *types.QsdevConfig, opts ValidateOptions) []ValidationError {
	if len(opts.MCPToolNames) == 0 {
		return nil
	}
	known := toSet(opts.MCPToolNames)
	var errs []ValidationError
	for _, t := range cfg.MCP.DisabledTools {
		if !known[t] {
			errs = append(errs, ValidationError{
				Field:   "mcp.disabled_tools",
				Value:   t,
				Message: "unknown MCP tool name; " + validValues(opts.MCPToolNames),
			})
		}
	}
	return errs
}

// validateProfiles checks each profile key against its own registry: the
// project-type `profile` against opts.ProfileNames (skipped when none are
// given) and `infra_profile` against the built-in infrastructure profiles.
func validateProfiles(cfg *types.QsdevConfig, opts ValidateOptions) []ValidationError {
	var errs []ValidationError
	if cfg.Profile != "" && len(opts.ProfileNames) > 0 && !slices.Contains(opts.ProfileNames, cfg.Profile) {
		errs = append(errs, ValidationError{
			Field:   "profile",
			Value:   cfg.Profile,
			Message: "unknown project-type profile; " + validValues(opts.ProfileNames),
		})
	}
	if cfg.InfraProfile != "" {
		infra := profile.DefaultProfileRegistry()
		if _, ok := infra.Get(cfg.InfraProfile); !ok {
			errs = append(errs, ValidationError{
				Field:   "infra_profile",
				Value:   cfg.InfraProfile,
				Message: "unknown infrastructure profile; " + validValues(infra.Names()),
			})
		}
	}
	return errs
}

// validateHooks checks the hooks block: each file-boundary extra read path
// must be one the hook can resolve and must not lift the read boundary.
func validateHooks(h types.HooksConfig) []ValidationError {
	var errs []ValidationError
	for i, p := range h.FileBoundary.ExtraReadPaths {
		if err := validation.CheckBoundaryReadPath(p); err != nil {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("hooks.file_boundary.extra_read_paths[%d]", i),
				Value:   p,
				Message: err.Error(),
			})
		}
	}
	return errs
}

// validValues renders the accepted values of an enumerated field, in the
// order its source (catalog or registry) lists them, for a validation message.
func validValues(values []string) string {
	return "valid values: " + strings.Join(values, ", ")
}

// validateSplicedValues checks the syntax of the free-form language and
// service values that generation splices into devenv.nix, some of them
// unquoted (e.g. pkgs.postgresql_<version>). The committed .qsdev.yaml is
// team-shared input, so a value that could end a Nix expression is rejected
// here as well as at the answers boundary (devinit.ValidateAnswers).
func validateSplicedValues(cfg *types.QsdevConfig) []ValidationError {
	var errs []ValidationError
	for i, lang := range cfg.Languages {
		if lang.Version != "" && !validation.IsValidVersionConstraint(lang.Version) {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("languages[%d].version", i),
				Value:   lang.Version,
				Message: "invalid version; only letters, digits, spaces and . _ - + * ^ ~ < > = ! | , / are allowed",
			})
		}
		if lang.PackageManager != "" && !validation.IsValidToken(lang.PackageManager) {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("languages[%d].package_manager", i),
				Value:   lang.PackageManager,
				Message: "invalid package manager; must be a single word of letters, digits, '.', '_' or '-'",
			})
		}
	}
	for i, pkg := range cfg.Packages {
		if !validation.IsValidNixAttrPath(pkg) {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("packages[%d]", i),
				Value:   pkg,
				Message: "invalid package; must be a Nix attribute path such as jq or python3Packages.black",
			})
		}
	}
	for i, svc := range cfg.Services {
		if svc.Version != "" && !validation.IsValidToken(svc.Version) {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("services[%d].version", i),
				Value:   svc.Version,
				Message: "invalid version; must be a single word of letters, digits, '.', '_' or '-'",
			})
		}
		for _, k := range slices.Sorted(maps.Keys(svc.Options)) {
			if !validation.IsValidEnvKey(k) || !validation.IsValidToken(svc.Options[k]) {
				errs = append(errs, ValidationError{
					Field:   fmt.Sprintf("services[%d].options.%s", i, k),
					Value:   svc.Options[k],
					Message: "invalid option; keys must be identifiers and values a single word of letters, digits, '.', '_' or '-'",
				})
			}
		}
	}
	return errs
}

// DefaultQsdevConfig returns a QsdevConfig with organization defaults.
func DefaultQsdevConfig() *types.QsdevConfig {
	t := true
	enabled := true
	return &types.QsdevConfig{
		Version: types.ConfigVersionCurrent,
		Security: types.SecurityConfig{
			Level:           "enhanced",
			AgeGating:       &t,
			ScriptBlocking:  &t,
			LockEnforcement: &t,
			VulnScanning:    &t,
		},
		ClaudeCode: types.ClaudeCodeConfig{
			Enabled:         &enabled,
			PermissionLevel: "standard",
		},
	}
}

func toSet(strs []string) map[string]bool {
	m := make(map[string]bool, len(strs))
	for _, s := range strs {
		m[s] = true
	}
	return m
}

// toInt converts a YAML-decoded value to int. YAML typically decodes
// integers as int, but we handle float64 for robustness.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		if n == float64(int(n)) {
			return int(n), true
		}
		return 0, false
	default:
		return 0, false
	}
}
