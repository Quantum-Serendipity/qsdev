package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	ccaddon "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
)

// Tools returns the five Claude Code tool registrations. Each handler delegates
// to a real generator (the P19 reference adapter or addons/claudecode) and reads
// the project root from its ToolCallContext; none is a stub. Missing
// prerequisites degrade to a structured not_configured error rather than crash.
func (a *Adapter) Tools() []spi.ToolRegistration {
	return []spi.ToolRegistration{
		{
			Name:        toolPermissions,
			Description: "Translate the project's qsdev permission preset into the concrete Claude Code allow/deny/ask rules (rendered .claude/settings.json), with a per-rule explanation of the targeted tool and whether each deny rule is a base or additional rule. Optionally filter to a single tool category.",
			InputSchema: optionalStringSchema("category", "Restrict the returned rules to those targeting this Claude Code tool (e.g. \"Bash\", \"Read\", \"WebFetch\"). Case-insensitive; omit for all rules."),
			Category:    middleware.CategoryPolicy,
			Tier:        tierStandard,
			Handler:     a.handlePermissions,
		},
		{
			Name:        toolHooks,
			Description: "Enumerate the Claude Code hooks deployed under <project>/.claude/hooks/: for each script the file path, mode, size, and last-modified time, cross-referenced with the deployed .claude/settings.json to report the event, matcher, command, and enforcement mode each script is wired to.",
			InputSchema: toolutil.EmptyObjectSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Handler:     a.handleHooks,
		},
		{
			Name:        toolContextBudget,
			Description: "Measure the token footprint of the project's Claude Code context files (CLAUDE.md and .claude/rules/*.md) against a model's context window, reporting total consumption, the remaining budget, and whether the project is within the 5% threshold. Defaults to the sonnet model.",
			InputSchema: optionalStringSchema("model", "Model whose context window to budget against: \"sonnet\" (200k) or \"opus\" (1M). Unknown values fall back to sonnet."),
			Category:    middleware.CategoryStatus,
			Tier:        tierExtended,
			Handler:     a.handleContextBudget,
		},
		{
			Name:        toolConfigRender,
			Description: "Render the Claude Code configuration files (.claude/settings.json and .mcp.json) from the project's qsdev policy. Dry-run by default: it returns the generated file contents without writing. Set write=true to materialize the files to disk.",
			InputSchema: optionalBoolSchema("write", "Write the rendered files to disk instead of returning them as a dry-run preview."),
			Category:    middleware.CategoryGeneral,
			Tier:        tierExtended,
			Handler:     a.handleConfigRender,
		},
		{
			Name:        toolEnforcementGaps,
			Description: "Report the enforcement gaps between the security isolation each active deny rule ideally requires (kernel-level sandboxing) and what Claude Code actually provides (PreToolUse hooks), with a mitigation for each gap.",
			InputSchema: toolutil.EmptyObjectSchema(),
			Category:    middleware.CategoryPolicy,
			Tier:        tierExtended,
			Handler:     a.handleEnforcementGaps,
		},
	}
}

// handlePermissions renders the real settings.json for the project's preset via
// the reference adapter's TranslatePermissions, then explains every allow/deny/
// ask rule. An optional category argument filters to one targeted tool.
func (a *Adapter) handlePermissions(ctx context.Context, cc *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	preset := presetFor(cc.ProjectRoot)
	arts, err := a.ref.TranslatePermissions(ctx, &aiframework.PermissionPolicy{Preset: preset})
	if err != nil {
		return toolutil.NotConfigured("could not render claude code permissions",
			map[string]any{"preset": preset, "error": err.Error()}), nil
	}
	settings, ok := settingsFromArtifacts(arts)
	if !ok {
		return toolutil.NotConfigured("permission translation produced no settings.json",
			map[string]any{"preset": preset}), nil
	}

	filter := stringArg(req.Arguments, "category")
	rules := explainPermissionRules(settings, filter)

	structured := map[string]any{
		"preset":      preset,
		"active_tier": arts.ActiveTier.String(),
		"allow_count": len(settings.Permissions.Allow),
		"deny_count":  len(settings.Permissions.Deny),
		"ask_count":   len(settings.Permissions.Ask),
		"rules":       rules,
	}
	if filter != "" {
		structured["category_filter"] = filter
	}
	text := fmt.Sprintf("claude code permissions (preset %q, tier %s): %d allow, %d deny, %d ask",
		preset, arts.ActiveTier.String(), len(settings.Permissions.Allow),
		len(settings.Permissions.Deny), len(settings.Permissions.Ask))
	return &spi.ToolResult{Text: text, Structured: structured}, nil
}

// handleHooks inspects the deployed hooks directory and cross-references the
// deployed settings.json wiring. A missing hooks directory degrades gracefully.
func (a *Adapter) handleHooks(_ context.Context, cc *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	hooksDir := filepath.Join(cc.ProjectRoot, ".claude", "hooks")
	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return toolutil.NotConfigured("no claude code hooks deployed",
				map[string]any{"hooks_dir": hooksDir}), nil
		}
		return nil, fmt.Errorf("reading hooks directory %s: %w", hooksDir, err)
	}

	wiring := deployedHookWiring(cc.ProjectRoot)
	hooks := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		entry := hookFileEntry(hooksDir, e, wiring[e.Name()])
		hooks = append(hooks, entry)
	}

	structured := map[string]any{
		"hooks_dir":     hooksDir,
		"count":         len(hooks),
		"settings_read": len(wiring) > 0,
		"hooks":         hooks,
	}
	text := fmt.Sprintf("%d claude code hook script(s) deployed in %s", len(hooks), hooksDir)
	return &spi.ToolResult{Text: text, Structured: structured}, nil
}

// handleContextBudget delegates to the addon's CalculateContextBudget, reading
// the project root from the call context.
func (a *Adapter) handleContextBudget(_ context.Context, cc *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	model := stringArg(req.Arguments, "model")
	if model == "" {
		model = ccaddon.ModelSonnet
	}
	budget, err := ccaddon.CalculateContextBudget(cc.ProjectRoot, model)
	if err != nil {
		return toolutil.NotConfigured("could not calculate context budget",
			map[string]any{"model": model, "error": err.Error()}), nil
	}

	remaining := budget.MaxTokens - budget.TotalTokens
	structured := map[string]any{
		"model":             budget.ModelSize,
		"max_tokens":        budget.MaxTokens,
		"total_tokens":      budget.TotalTokens,
		"claude_md_tokens":  budget.ClaudeMdTokens,
		"rules_tokens":      budget.RulesTokens,
		"skill_desc_tokens": budget.SkillDescTokens,
		"remaining_tokens":  remaining,
		"budget_pct":        budget.BudgetPct,
		"within_budget":     budget.Validate() == nil,
	}
	text := fmt.Sprintf("context budget (%s): %d/%d tokens (%.2f%%), %d remaining",
		budget.ModelSize, budget.TotalTokens, budget.MaxTokens, budget.BudgetPct, remaining)
	return &spi.ToolResult{Text: text, Structured: structured}, nil
}

// handleConfigRender renders the Claude Code config via the reference adapter.
// It is dry-run by default (returns contents without writing); write=true
// materializes the files through the generation pipeline.
func (a *Adapter) handleConfigRender(ctx context.Context, cc *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	input := a.policyInputFor(cc.ProjectRoot)
	files, err := a.ref.Render(ctx, input)
	if err != nil {
		return toolutil.NotConfigured("could not render claude code configuration",
			map[string]any{"preset": input.Permissions.Preset, "error": err.Error()}), nil
	}

	rendered := make([]map[string]any, 0, len(files))
	for _, f := range files {
		rendered = append(rendered, map[string]any{
			"path":     f.Path,
			"mode":     f.Mode.String(),
			"strategy": f.Strategy.String(),
			"bytes":    len(f.Content),
			"content":  string(f.Content),
		})
	}

	write := boolArg(req.Arguments, "write")
	structured := map[string]any{
		"project_root":      cc.ProjectRoot,
		"preset":            input.Permissions.Preset,
		"write":             write,
		"file_count":        len(files),
		"files":             rendered,
		"validation_issues": validationIssues(a.ref.Validate(ctx, files)),
	}

	if write {
		res, werr := generate.WriteFiles(files, generate.PipelineOptions{
			ProjectRoot:       cc.ProjectRoot,
			SectionMergeFunc:  merge.SectionMarkers,
			ThreeWayMergeFunc: merge.MergeOnCreate,
		})
		if werr != nil {
			return nil, fmt.Errorf("writing rendered claude code files: %w", werr)
		}
		structured["write_result"] = map[string]any{
			"created": res.Created, "updated": res.Updated,
			"skipped": res.Skipped, "failed": res.Failed,
			"summary": res.Summary(),
		}
		text := fmt.Sprintf("rendered and wrote %d claude code file(s): %s", len(files), res.Summary())
		return &spi.ToolResult{Text: text, Structured: structured, IsError: res.HasFailures()}, nil
	}

	text := fmt.Sprintf("rendered %d claude code file(s) (dry-run, not written)", len(files))
	return &spi.ToolResult{Text: text, Structured: structured}, nil
}

// handleEnforcementGaps renders the project's active deny rules and reports, via
// the reference adapter, the gap between the kernel-level isolation each rule
// ideally requires and the hook-level enforcement Claude Code provides.
func (a *Adapter) handleEnforcementGaps(ctx context.Context, cc *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	preset := presetFor(cc.ProjectRoot)
	arts, err := a.ref.TranslatePermissions(ctx, &aiframework.PermissionPolicy{Preset: preset})
	if err != nil {
		return toolutil.NotConfigured("could not render claude code permissions for gap analysis",
			map[string]any{"preset": preset, "error": err.Error()}), nil
	}
	settings, ok := settingsFromArtifacts(arts)
	if !ok {
		return toolutil.NotConfigured("permission translation produced no settings.json",
			map[string]any{"preset": preset}), nil
	}

	policy := &aiframework.PermissionPolicy{Preset: preset, DenyRules: denyRulesFromSettings(settings)}
	gaps := a.ref.ReportGaps(ctx, policy)

	formatted := make([]map[string]any, 0, len(gaps))
	for _, g := range gaps {
		formatted = append(formatted, map[string]any{
			"rule":          g.Rule.Pattern,
			"reason":        g.Rule.Reason,
			"required_tier": g.RequiredTier.String(),
			"actual_tier":   g.ActualTier.String(),
			"description":   g.Description,
			"mitigation":    g.Mitigation,
		})
	}

	structured := map[string]any{
		"preset":        preset,
		"enforced_tier": arts.ActiveTier.String(),
		"gap_count":     len(gaps),
		"gaps":          formatted,
	}
	text := fmt.Sprintf("claude code enforcement gaps (preset %q): %d deny rule(s) enforced at %s tier, all wanting kernel-level isolation",
		preset, len(gaps), arts.ActiveTier.String())
	return &spi.ToolResult{Text: text, Structured: structured}, nil
}

// explainPermissionRules classifies every allow/deny/ask rule in the rendered
// settings into a structured explanation (kind, targeted tool, source), filtered
// to a single tool category when filter is non-empty.
func explainPermissionRules(settings *ccaddon.SettingsJSON, filter string) []map[string]any {
	base := baseDenySet()
	var rules []map[string]any
	add := func(kind string, patterns []string, source func(string) string) {
		for _, p := range patterns {
			tool := toolForPattern(p)
			if filter != "" && !strings.EqualFold(tool, filter) {
				continue
			}
			rules = append(rules, map[string]any{
				"rule": p, "kind": kind, "tool": tool, "source": source(p),
			})
		}
	}
	add("deny", settings.Permissions.Deny, func(p string) string {
		if base[p] {
			return "base"
		}
		return "additional"
	})
	add("allow", settings.Permissions.Allow, func(string) string { return "preset" })
	add("ask", settings.Permissions.Ask, func(string) string { return "preset" })
	return rules
}

// hookFileEntry builds the structured record for one deployed hook script,
// merging filesystem metadata with the settings.json wiring (which may be empty
// when the script is present on disk but not wired into settings).
func hookFileEntry(dir string, e os.DirEntry, wires []hookWiring) map[string]any {
	path := filepath.Join(dir, e.Name())
	entry := map[string]any{
		"name":       e.Name(),
		"path":       path,
		"executable": false,
	}
	if info, err := e.Info(); err == nil {
		entry["mode"] = info.Mode().String()
		entry["size"] = info.Size()
		entry["modified"] = info.ModTime().UTC().Format("2006-01-02T15:04:05Z")
		entry["executable"] = info.Mode().Perm()&0o111 != 0
	}
	bindings := make([]map[string]any, 0, len(wires))
	for _, w := range wires {
		bindings = append(bindings, map[string]any{
			"event":       w.Event,
			"matcher":     w.Matcher,
			"command":     w.Command,
			"timeout":     w.Timeout,
			"enforcement": enforcementMode(w.Event),
		})
	}
	entry["wired"] = len(bindings) > 0
	entry["bindings"] = bindings
	return entry
}

// enforcementMode describes how a hook event participates in enforcement. Claude
// Code hooks signal via exit code; PreToolUse runs before the tool and can deny
// it (blocking), while later events only observe.
func enforcementMode(event string) string {
	if event == "PreToolUse" {
		return "blocking"
	}
	return "observing"
}

// validationIssues converts reference-adapter validation issues into structured
// records.
func validationIssues(issues []aiframework.ValidationIssue) []map[string]any {
	out := make([]map[string]any, 0, len(issues))
	for _, i := range issues {
		out = append(out, map[string]any{
			"path": i.Path, "message": i.Message, "severity": i.Severity.String(),
		})
	}
	return out
}

// baseDenySet returns the set of base deny-rule patterns for source classifica-
// tion. A nil/empty catalog yields an empty set (everything reads as additional).
func baseDenySet() map[string]bool {
	set := make(map[string]bool)
	for _, r := range ccaddon.AllBaseDenyRules() {
		set[r] = true
	}
	return set
}

// denyRulesFromSettings converts a rendered deny list into permission rules,
// preserving order, so the gap analyzer sees the project's active deny policy.
func denyRulesFromSettings(settings *ccaddon.SettingsJSON) []aiframework.PermissionRule {
	rules := make([]aiframework.PermissionRule, 0, len(settings.Permissions.Deny))
	for _, p := range settings.Permissions.Deny {
		rules = append(rules, aiframework.PermissionRule{Pattern: p})
	}
	return rules
}

// settingsFromArtifacts extracts the rendered SettingsJSON from permission
// artifacts. It returns false when no settings.json file is present or the
// content is not parseable.
func settingsFromArtifacts(arts *aiframework.PermissionArtifacts) (*ccaddon.SettingsJSON, bool) {
	if arts == nil {
		return nil, false
	}
	for _, f := range arts.GeneratedFiles {
		if filepath.Base(f.Path) != "settings.json" {
			continue
		}
		var s ccaddon.SettingsJSON
		if err := json.Unmarshal(f.Content, &s); err != nil {
			return nil, false
		}
		return &s, true
	}
	return nil, false
}

// hookWiring is a single event binding of a hook script taken from settings.json.
type hookWiring struct {
	Event   string
	Matcher string
	Command string
	Timeout int
}

// deployedHookWiring reads the project's deployed .claude/settings.json and maps
// each hook script's basename to the event bindings that reference it. It
// returns an empty map when settings.json is absent or unparseable.
func deployedHookWiring(projectRoot string) map[string][]hookWiring {
	wiring := make(map[string][]hookWiring)
	data, err := os.ReadFile(filepath.Join(projectRoot, ".claude", "settings.json"))
	if err != nil {
		return wiring
	}
	var settings ccaddon.SettingsJSON
	if err := json.Unmarshal(data, &settings); err != nil {
		return wiring
	}
	for event, matchers := range settings.Hooks {
		for _, m := range matchers {
			for _, h := range m.Hooks {
				name := hookScriptName(h.Command)
				if name == "" {
					continue
				}
				wiring[name] = append(wiring[name], hookWiring{
					Event: event, Matcher: m.Matcher, Command: h.Command, Timeout: h.Timeout,
				})
			}
		}
	}
	for name := range wiring {
		sort.Slice(wiring[name], func(i, j int) bool {
			return wiring[name][i].Event < wiring[name][j].Event
		})
	}
	return wiring
}

// hookScriptName extracts the deployed script basename a hook command references
// (the token after ".claude/hooks/", up to the first space). It returns "" for
// commands that do not invoke a script under .claude/hooks/ (e.g. "qsdev
// selfprotect").
func hookScriptName(command string) string {
	const marker = ".claude/hooks/"
	idx := strings.Index(command, marker)
	if idx < 0 {
		return ""
	}
	rest := command[idx+len(marker):]
	if cut := strings.IndexAny(rest, " \t\"'"); cut >= 0 {
		rest = rest[:cut]
	}
	return rest
}

// toolForPattern returns the Claude Code tool a permission pattern targets: the
// token before the first "(", or the whole pattern when it has no arguments.
func toolForPattern(pattern string) string {
	if i := strings.IndexByte(pattern, '('); i >= 0 {
		return pattern[:i]
	}
	return pattern
}
