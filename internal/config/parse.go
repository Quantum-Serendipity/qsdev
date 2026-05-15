// Package config provides parsing, validation, and migration for .qsdev.yaml
// project configuration files.
package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Quantum-Serendipity/qsdev/internal/validation"
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
	ProfileNames []string
	ToolNames    []string
}

// ParseQsdevConfig reads and parses a .qsdev.yaml file at path.
//
// It uses two-pass parsing: first unmarshal to map[string]any to extract
// and validate the version field with clear error messages, then full
// struct unmarshal into QsdevConfig. Unknown YAML fields are silently
// ignored (gopkg.in/yaml.v3 default behavior).
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
		return nil, fmt.Errorf("missing required field \"version\" in .qsdev.yaml; add \"version: %d\" at the top of the file",
			types.ConfigVersionCurrent)
	}

	versionInt, ok := toInt(versionRaw)
	if !ok {
		return nil, fmt.Errorf("field \"version\" must be an integer, got %T", versionRaw)
	}

	if versionInt > types.ConfigVersionMax {
		return nil, fmt.Errorf(
			"config version %d is newer than this binary supports (max %d); please update qsdev to the latest version",
			versionInt, types.ConfigVersionMax)
	}

	if versionInt < types.ConfigVersionMin {
		return nil, fmt.Errorf(
			"config version %d is no longer supported (minimum %d); run \"qsdev config migrate\" to upgrade",
			versionInt, types.ConfigVersionMin)
	}

	// Pass 2: full struct unmarshal.
	var cfg types.QsdevConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
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

	// Validate security level.
	if cfg.Security.Level != "" && !validation.IsValidSecurityLevel(cfg.Security.Level) {
		errs = append(errs, ValidationError{
			Field:   "security.level",
			Value:   cfg.Security.Level,
			Message: "invalid security level; valid values: baseline, enhanced, strict",
		})
	}

	// Validate claude_code.permission_level.
	if cfg.ClaudeCode.PermissionLevel != "" && !validation.IsValidPermissionPreset(cfg.ClaudeCode.PermissionLevel) {
		errs = append(errs, ValidationError{
			Field:   "claude_code.permission_level",
			Value:   cfg.ClaudeCode.PermissionLevel,
			Message: "invalid permission level; valid values: minimal, standard, permissive, custom",
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
		for _, t := range cfg.Tools.Disabled {
			if !knownTools[t] {
				errs = append(errs, ValidationError{
					Field:   "tools.disabled",
					Value:   t,
					Message: "unknown tool name",
				})
			}
		}
	}

	// Validate profile against known profile names.
	if cfg.Profile != "" && len(opts.ProfileNames) > 0 {
		knownProfiles := toSet(opts.ProfileNames)
		if !knownProfiles[cfg.Profile] {
			errs = append(errs, ValidationError{
				Field:   "profile",
				Value:   cfg.Profile,
				Message: "unknown profile name",
			})
		}
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
				Message: "invalid security level; valid values: baseline, enhanced, strict",
			})
		}
		if cfg.Client.DataClassification != "" && !validation.IsValidDataClassification(cfg.Client.DataClassification) {
			errs = append(errs, ValidationError{
				Field:   "client.data_classification",
				Value:   cfg.Client.DataClassification,
				Message: "invalid data classification; valid values: public, internal, confidential",
			})
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
			Level:          "enhanced",
			AgeGating:      &t,
			ScriptBlocking: &t,
			LockEnforce:    &t,
			VulnScanning:   &t,
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
