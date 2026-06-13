package check

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// RegenerateFunc produces a fresh set of generated files from saved project
// answers. Injected by the command layer to avoid circular imports.
type RegenerateFunc func(projectRoot string) (map[string]types.GeneratedFile, error)

// ApplyAutoFixes attempts to fix each AutoFixable result. It returns an
// updated copy of the results slice with fixed results changed to StatusPass.
//
// regenerate is an optional callback for restoring deleted generated files.
// Pass nil when regeneration is not available.
func ApplyAutoFixes(results []CheckResult, projectRoot string, regenerate RegenerateFunc) []CheckResult {
	updated := make([]CheckResult, len(results))
	copy(updated, results)

	// Collect missing deny rules.
	var missingRules []string
	var ruleIndices []int
	for i, r := range updated {
		if r.AutoFixable && r.Name == "deny_rule_missing" {
			if rule, ok := r.Metadata["rule"]; ok {
				missingRules = append(missingRules, rule)
				ruleIndices = append(ruleIndices, i)
			}
		}
	}

	if len(missingRules) > 0 {
		if err := fixDenyRules(projectRoot, missingRules); err == nil {
			for _, idx := range ruleIndices {
				updated[idx].Status = StatusPass
				updated[idx].Message = "Auto-fixed: " + updated[idx].Message
				updated[idx].AutoFixable = false
			}
		}
	}

	updated = fixDeletedFiles(updated, projectRoot, regenerate)

	return updated
}

// fixDeletedFiles restores deleted generated files using the regenerate callback.
func fixDeletedFiles(results []CheckResult, projectRoot string, regenerate RegenerateFunc) []CheckResult {
	if regenerate == nil {
		return results
	}

	var deletedIndices []int
	for i, r := range results {
		if r.AutoFixable && strings.HasPrefix(r.Name, "file_exists_") {
			if _, ok := r.Metadata["file"]; ok {
				deletedIndices = append(deletedIndices, i)
			}
		}
	}
	if len(deletedIndices) == 0 {
		return results
	}

	freshFiles, err := regenerate(projectRoot)
	if err != nil {
		slog.Warn("auto-fix: regeneration failed", "error", err)
		return results
	}

	for _, idx := range deletedIndices {
		relPath := results[idx].Metadata["file"]
		fresh, ok := freshFiles[relPath]
		if !ok {
			continue
		}

		absPath := filepath.Join(projectRoot, relPath)
		mode := fresh.Mode
		if mode == 0 {
			mode = fileutil.ModeReadWrite
		}
		if err := fileutil.WriteFileAtomic(absPath, fresh.Content, mode); err != nil {
			continue
		}

		results[idx].Status = StatusPass
		results[idx].Message = "Auto-fixed: " + results[idx].Message
		results[idx].AutoFixable = false
	}
	return results
}

// fixDenyRules reads .claude/settings.json, adds missing deny rules, and
// writes it back.
func fixDenyRules(projectRoot string, missingRules []string) error {
	settingsPath := filepath.Join(projectRoot, ".claude", "settings.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("reading settings.json: %w", err)
	}

	// Parse into a generic map to preserve unknown fields.
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parsing settings.json: %w", err)
	}

	// Navigate to permissions.deny.
	perms, ok := settings["permissions"]
	if !ok {
		perms = map[string]any{}
		settings["permissions"] = perms
	}
	permsMap, ok := perms.(map[string]any)
	if !ok {
		return fmt.Errorf("permissions is not an object")
	}

	var existingDeny []any
	if d, ok := permsMap["deny"]; ok {
		existingDeny, _ = d.([]any)
	}

	// Build set of existing rules.
	existing := make(map[string]bool, len(existingDeny))
	for _, r := range existingDeny {
		if s, ok := r.(string); ok {
			existing[s] = true
		}
	}

	// Add missing rules.
	for _, rule := range missingRules {
		if !existing[rule] {
			existingDeny = append(existingDeny, rule)
		}
	}
	permsMap["deny"] = existingDeny

	// Write back.
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings.json: %w", err)
	}
	out = append(out, '\n')

	if err := fileutil.WriteFileAtomic(settingsPath, out, fileutil.ModeReadWrite); err != nil {
		return fmt.Errorf("writing settings.json: %w", err)
	}

	return nil
}
