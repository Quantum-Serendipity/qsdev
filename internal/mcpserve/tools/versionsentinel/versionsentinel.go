// Package versionsentinel implements the version-sentinel MCP tool module for
// the universal qsdev server: check_versions (the dependency versions the
// project's manifests declare), detect_drift (lock-file entries that disagree
// with their manifest), manifest_coverage (which of the project's manifests
// Version-Sentinel version-diffs, which it only checks for a lockfile and
// which it does not check) and version_history (the recorded version-change
// events).
//
// The tools run behind the universal server's middleware chain (guardrail,
// rate limiting, content safety, audit) like every other mounted tool, and read
// only inside the server's resolved project root: a caller-supplied path that
// escapes it (via "..", an absolute path elsewhere, or a symlink) is refused.
package versionsentinel

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/internal/vsentinel"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// ModuleName names this tool module. It is the value `qsdev mcp serve
// --module` takes and the key of the catalog's version-sentinel MCP server.
const ModuleName = "version-sentinel"

// tierStandard mirrors the other tool modules' standard tier.
const tierStandard = 1

// module holds the project root every tool is confined to and the ecosystem
// registry whose modules name the manifests to report on.
type module struct {
	projectRoot string
	registry    *ecosystem.Registry
}

// Tools returns the version-sentinel tool registrations bound to projectRoot.
func Tools(projectRoot string) []spi.ToolRegistration {
	m := &module{projectRoot: projectRoot, registry: ecosystem.DefaultRegistry()}
	return []spi.ToolRegistration{
		{
			Name:        "check_versions",
			Description: "Scan manifest files and report current dependency versions.",
			InputSchema: rootArgSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Annotations: spi.ReadOnlyAnnotations(false),
			Handler:     m.checkVersions,
		},
		{
			Name: "detect_drift",
			Description: "Compare lockfile entries against manifest declarations to detect version drift in the given directory. " +
				"Each manifest carries verification \"diffed\" (versions compared with the lockfile) or \"presence-only\" (only lockfile presence checked). " +
				"coverage is \"complete\" only when every manifest was diffed, \"none\" when no manifest was found, and otherwise \"partial\", with not_verified naming each manifest not version-checked: presence-only ones, manifests of ecosystems without a drift checker, and manifests in subdirectories (pass project_root to check one). " +
				subdirWalkNote +
				"A partial result where earlier versions reported zero drift is a correction (those manifests were never diffed), not a regression.",
			InputSchema: rootArgSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Annotations: spi.ReadOnlyAnnotations(false),
			Handler:     m.detectDrift,
		},
		{
			Name: "manifest_coverage",
			Description: "Report, for the languages in the project's qsdev answers, which manifests Version-Sentinel version-diffs against their lockfile (diffed) and which it only checks for lockfile presence (presence_only). " +
				"coverage is \"complete\" only when every manifest is diffed, \"none\" when the languages declare no manifest, and otherwise \"partial\", with not_version_checked naming each presence-only manifest. " +
				"A manifest earlier versions reported as covered but now listed as presence_only is a correction (its versions were never compared), not a regression. " +
				"The generated CLAUDE.md Version-Sentinel section and .version-sentinel/ignore state this same classification.",
			InputSchema: toolutil.EmptyObjectSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Annotations: spi.ReadOnlyAnnotations(false),
			Handler:     m.manifestCoverage,
		},
		{
			Name:        "version_history",
			Description: "Read the version-sentinel event log showing a timeline of version changes.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"log_path": map[string]any{
						"type":        "string",
						"description": "Path to events.jsonl within the project root (defaults to .version-sentinel/events.jsonl)",
					},
				},
			},
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Annotations: spi.ReadOnlyAnnotations(false),
			Handler:     m.versionHistory,
		},
	}
}

// subdirWalkNote states which subdirectories detect_drift looks in for
// unchecked manifests, so the tool does not over-claim what it examined.
var subdirWalkNote = fmt.Sprintf("Subdirectories are searched up to %d levels deep; hidden, node_modules and vendor directories, symlinked directories and unreadable directories are not searched. ", ecosystem.ProjectScanDepth)

// rootArgSchema is the input schema of the tools that scan a directory.
func rootArgSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_root": map[string]any{
				"type":        "string",
				"description": "Directory to scan, within the project root (defaults to the project root)",
			},
		},
	}
}

// confinedPath resolves the caller-supplied path argument key against the
// project root. It returns "" when the argument is absent, and a denial result
// when the path escapes the root.
func (m *module) confinedPath(args map[string]any, key string) (string, *spi.ToolResult) {
	arg := toolutil.StringArgOr(args, key, "")
	if arg == "" {
		return "", nil
	}
	resolved, ok := toolutil.ConfineToRoot(m.projectRoot, arg)
	if !ok {
		return "", toolutil.Denied(fmt.Sprintf("%s %q is outside the project root %q", key, arg, m.projectRoot), map[string]any{key: arg})
	}
	return resolved, nil
}

// scanRoot returns the directory to scan: the caller's project_root, confined
// to the project root, or the project root itself.
func (m *module) scanRoot(args map[string]any) (string, *spi.ToolResult) {
	dir, denied := m.confinedPath(args, "project_root")
	if denied != nil || dir != "" {
		return dir, denied
	}
	return m.projectRoot, nil
}

func (m *module) checkVersions(_ context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	root, denied := m.scanRoot(req.Arguments)
	if denied != nil {
		return denied, nil
	}
	report, err := vsentinel.CheckVersions(root)
	if err != nil {
		return toolutil.ErrorResult(err.Error(), nil), nil
	}
	return toolutil.Result(toolutil.MarshalText(report, "version report"), report), nil
}

func (m *module) detectDrift(_ context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	root, denied := m.scanRoot(req.Arguments)
	if denied != nil {
		return denied, nil
	}
	report, err := vsentinel.DetectDrift(root, m.knownManifests()...)
	if err != nil {
		return toolutil.ErrorResult(err.Error(), nil), nil
	}
	return toolutil.Result(toolutil.MarshalText(report, "drift report"), report), nil
}

// manifestCoverage reports the coverage of the languages recorded in the
// project's primary qsdev answers file.
func (m *module) manifestCoverage(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	a, err := answers.RequirePrimary(m.projectRoot)
	if err != nil {
		return toolutil.NotConfigured(fmt.Sprintf("loading qsdev answers: %v", err), nil), nil
	}
	report := classifyCoverage(ecosystem.LanguageManifestCoverage(a.Languages, m.registry).AllManifests)
	return toolutil.Result(toolutil.MarshalText(report, "manifest coverage"), report), nil
}

// knownManifests returns the base names or globs of every manifest a
// registered ecosystem module declares — under its default configuration and,
// when the project has qsdev answers, under the project's own configuration —
// so detect_drift can name manifests it has no checker for.
func (m *module) knownManifests() []string {
	defaultConfig := func(ecosystem.EcosystemModule) ecosystem.ModuleConfig { return ecosystem.ModuleConfig{} }
	infos := ecosystem.AggregateManifestCoverage(m.registry.All(), defaultConfig).AllManifests
	if a, err := answers.RequirePrimary(m.projectRoot); err == nil {
		infos = append(infos, ecosystem.LanguageManifestCoverage(a.Languages, m.registry).AllManifests...)
	}
	out := make([]string, 0, len(infos))
	for _, info := range infos {
		out = append(out, path.Base(filepath.ToSlash(info.Path)))
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// manifestCoverageReport is the manifest_coverage result. Coverage is
// complete only when every manifest is version-diffed, and none when the
// project's languages declare no manifest.
type manifestCoverageReport struct {
	Diffed            []manifestCoverageEntry `json:"diffed"`
	PresenceOnly      []manifestCoverageEntry `json:"presence_only"`
	Uncovered         []manifestCoverageEntry `json:"uncovered"`
	Coverage          string                  `json:"coverage"`
	NotVersionChecked []string                `json:"not_version_checked,omitempty"`
}

// manifestCoverageEntry is one manifest a language module declares.
type manifestCoverageEntry struct {
	Path      string `json:"path"`
	Ecosystem string `json:"ecosystem"`
	LockFile  string `json:"lock_file,omitempty"`
}

// classifyCoverage builds the report from vsentinel.ClassifyManifests, the
// classification the generated CLAUDE.md section also uses. The modules'
// VSSupported flag is deliberately ignored: it does not reflect which
// manifests have a drift checker.
func classifyCoverage(manifests []ecosystem.ManifestFileInfo) manifestCoverageReport {
	c := vsentinel.ClassifyManifests(manifests)
	report := manifestCoverageReport{
		Diffed:       coverageEntries(c.Diffed),
		PresenceOnly: coverageEntries(c.PresenceOnly),
		Uncovered:    coverageEntries(c.Uncovered),
		Coverage:     vsentinel.CoverageComplete,
	}
	for _, m := range c.NotDiffed {
		reason := "no drift check: neither versions nor lockfile presence verified"
		if slices.Contains(c.PresenceOnly, m) {
			reason = "lockfile presence only, versions not compared"
		}
		report.Coverage = vsentinel.CoveragePartial
		report.NotVersionChecked = append(report.NotVersionChecked,
			fmt.Sprintf("%s (%s: %s)", m.Path, m.Ecosystem, reason))
	}
	if len(manifests) == 0 {
		report.Coverage = vsentinel.CoverageNone
	}
	return report
}

// coverageEntries converts manifests to report entries, never nil.
func coverageEntries(manifests []ecosystem.ManifestFileInfo) []manifestCoverageEntry {
	out := make([]manifestCoverageEntry, 0, len(manifests))
	for _, m := range manifests {
		out = append(out, manifestCoverageEntry{Path: m.Path, Ecosystem: m.Ecosystem, LockFile: m.LockFile})
	}
	return out
}

func (m *module) versionHistory(_ context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	logPath, denied := m.confinedPath(req.Arguments, "log_path")
	if denied != nil {
		return denied, nil
	}
	if logPath == "" {
		logPath = filepath.Join(m.projectRoot, ".version-sentinel", "events.jsonl")
	}
	events, err := vsentinel.ReadVersionHistory(logPath)
	if err != nil {
		return toolutil.ErrorResult(err.Error(), nil), nil
	}
	if events == nil {
		events = []vsentinel.VersionEvent{}
	}
	return toolutil.Result(toolutil.MarshalText(events, "[]"), map[string]any{"events": events}), nil
}
