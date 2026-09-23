package check

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// RegenerateFunc produces a fresh set of generated files from saved project
// answers. Injected by the command layer to avoid circular imports.
type RegenerateFunc func(projectRoot string) (map[string]types.GeneratedFile, error)

// ApplyAutoFixes attempts to fix each AutoFixable result. It returns an
// updated copy of the results slice with fixed results changed to StatusPass.
// A fix that fails leaves its result failed and appends the reason to its
// message.
//
// regenerate is an optional callback for restoring deleted generated files.
// Pass nil when regeneration is not available. Restored files are recorded in
// the generation state at stateFile (skipped when stateFile is empty) so the
// next check compares against the restored content.
func ApplyAutoFixes(results []CheckResult, projectRoot, stateFile string, regenerate RegenerateFunc) []CheckResult {
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
		err := fixDenyRules(projectRoot, missingRules)
		if err != nil {
			slog.Warn("auto-fix: adding deny rules failed", "error", err)
		}
		for _, idx := range ruleIndices {
			markFixResult(&updated[idx], err)
		}
	}

	updated = fixDeletedFiles(updated, projectRoot, stateFile, regenerate)

	return updated
}

// markFixResult records the outcome of an auto-fix attempt on a result.
func markFixResult(r *CheckResult, err error) {
	if err != nil {
		r.Message = fmt.Sprintf("%s (auto-fix failed: %v)", r.Message, err)
		return
	}
	r.Status = StatusPass
	r.Message = "Auto-fixed: " + r.Message
	r.AutoFixable = false
}

// fixDeletedFiles restores deleted generated files using the regenerate callback.
func fixDeletedFiles(results []CheckResult, projectRoot, stateFile string, regenerate RegenerateFunc) []CheckResult {
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
		for _, idx := range deletedIndices {
			markFixResult(&results[idx], fmt.Errorf("regenerating files: %w", err))
		}
		return results
	}

	var restored []types.GeneratedFile
	for _, idx := range deletedIndices {
		relPath := results[idx].Metadata["file"]
		fresh, ok := freshFiles[relPath]
		if !ok {
			markFixResult(&results[idx], fmt.Errorf("%s is no longer generated for this project", relPath))
			continue
		}

		mode := fresh.Mode
		if mode == 0 {
			mode = fileutil.ModeReadWrite
		}
		fresh.Path = relPath
		fresh.Mode = mode
		if err := generate.WriteGeneratedFile(projectRoot, fresh); err != nil {
			slog.Warn("auto-fix: restoring file failed", "file", relPath, "error", err)
			markFixResult(&results[idx], err)
			continue
		}

		restored = append(restored, fresh)
		markFixResult(&results[idx], nil)
	}

	if err := recordRestoredFiles(stateFile, restored); err != nil {
		slog.Warn("auto-fix: recording restored files in state failed", "error", err)
	}
	return results
}

// recordRestoredFiles updates the generation state entries of restored files
// (hash, strategy, mode, owner and three-way-merge base) so the state matches
// what is now on disk.
func recordRestoredFiles(stateFile string, restored []types.GeneratedFile) error {
	if stateFile == "" || len(restored) == 0 {
		return nil
	}
	genState, err := state.LoadStateFromFile(stateFile)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}
	maps.Copy(genState.Files, state.RecordFiles(restored).Files)
	if err := state.SaveStateToFile(stateFile, genState); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}
	return nil
}

// fixDenyRules reads .claude/settings.json, adds missing deny rules, and
// writes it back. Only permissions.deny is changed: every other key keeps its
// position and its original encoding.
func fixDenyRules(projectRoot string, missingRules []string) error {
	settingsPath := filepath.Join(projectRoot, filepath.FromSlash(ClaudeSettingsRelPath))

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("reading settings.json: %w", err)
	}

	out, err := addDenyRules(data, missingRules)
	if err != nil {
		return fmt.Errorf("updating settings.json: %w", err)
	}

	if err := fileutil.WriteFileAtomic(settingsPath, out, fileutil.ModeReadWrite); err != nil {
		return fmt.Errorf("writing settings.json: %w", err)
	}

	return nil
}

// addDenyRules returns settings JSON with rules appended to permissions.deny
// (creating either as needed), preserving the order and raw encoding of every
// other member. A permissions or deny value of the wrong type is an error
// rather than being replaced.
func addDenyRules(data []byte, rules []string) ([]byte, error) {
	top, err := decodeOrderedObject(data)
	if err != nil {
		return nil, fmt.Errorf("parsing settings: %w", err)
	}

	var perms orderedObject
	if raw, ok := top.get("permissions"); ok {
		if perms, err = decodeOrderedObject(raw); err != nil {
			return nil, fmt.Errorf("permissions is not an object: %w", err)
		}
	}

	var deny []json.RawMessage
	if raw, ok := perms.get("deny"); ok {
		if err := json.Unmarshal(raw, &deny); err != nil {
			return nil, errors.New("permissions.deny is not an array")
		}
	}

	existing := make(map[string]bool, len(deny))
	for _, raw := range deny {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			existing[s] = true
		}
	}
	for _, rule := range rules {
		if existing[rule] {
			continue
		}
		existing[rule] = true
		enc, err := marshalNoEscape(rule)
		if err != nil {
			return nil, err
		}
		deny = append(deny, enc)
	}

	if deny == nil {
		deny = []json.RawMessage{}
	}
	denyRaw, err := marshalNoEscape(deny)
	if err != nil {
		return nil, err
	}
	perms.set("deny", denyRaw)

	permsRaw, err := perms.marshal()
	if err != nil {
		return nil, err
	}
	top.set("permissions", permsRaw)

	compact, err := top.marshal()
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, compact, "", "  "); err != nil {
		return nil, fmt.Errorf("formatting settings: %w", err)
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

// orderedObject is a JSON object whose members keep their original order and
// raw encoding.
type orderedObject struct {
	keys   []string
	values map[string]json.RawMessage
}

func (o *orderedObject) get(key string) (json.RawMessage, bool) {
	v, ok := o.values[key]
	return v, ok
}

func (o *orderedObject) set(key string, value json.RawMessage) {
	if o.values == nil {
		o.values = make(map[string]json.RawMessage)
	}
	if _, ok := o.values[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

func (o *orderedObject) marshal() (json.RawMessage, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		key, err := marshalNoEscape(k)
		if err != nil {
			return nil, err
		}
		b.Write(key)
		b.WriteByte(':')
		b.Write(o.values[k])
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// decodeOrderedObject decodes a JSON object, keeping member order and raw
// values.
func decodeOrderedObject(data []byte) (orderedObject, error) {
	var obj orderedObject
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return obj, fmt.Errorf("reading JSON: %w", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return obj, errors.New("expected a JSON object")
	}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return obj, fmt.Errorf("reading key: %w", err)
		}
		key, ok := keyTok.(string)
		if !ok {
			return obj, errors.New("expected an object key")
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return obj, fmt.Errorf("reading value of %q: %w", key, err)
		}
		obj.set(key, raw)
	}
	if _, err := dec.Token(); err != nil {
		return obj, fmt.Errorf("reading end of object: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return obj, errors.New("unexpected data after JSON object")
	}
	return obj, nil
}

// marshalNoEscape encodes v as JSON without HTML-escaping <, > and &, which
// are common in shell-command permission rules.
func marshalNoEscape(v any) (json.RawMessage, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("encoding %v: %w", v, err)
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}
