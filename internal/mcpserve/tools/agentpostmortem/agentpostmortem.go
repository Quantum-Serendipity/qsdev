// Package agentpostmortem implements the agent-postmortem MCP tool module for
// the universal qsdev server: analyze_session (one Claude Code session
// transcript), list_failure_patterns (failure patterns aggregated across
// transcripts) and generate_verification_checklist (the project's build, test
// and lint commands).
//
// The tools run behind the universal server's middleware chain (guardrail,
// rate limiting, content safety, audit) like every other mounted tool. The
// transcript tools only read .jsonl files inside the Claude Code projects
// directory ($CLAUDE_CONFIG_DIR/projects, or ~/.claude/projects): a path the
// agent supplies is resolved, symlinks included, and refused when it leaves
// that directory.
package agentpostmortem

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/internal/postmortem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// ModuleName names this tool module. It is the value `qsdev mcp serve
// --module` takes and the key of the catalog's agent-postmortem MCP server.
const ModuleName = "agent-postmortem"

// maxSessionFiles bounds how many transcripts one list_failure_patterns call
// parses, so a huge sessions directory cannot stall the MCP server.
const maxSessionFiles = 5000

// sessionExt is the extension of a Claude Code session transcript.
const sessionExt = ".jsonl"

// tierStandard mirrors the other tool modules' standard tier.
const tierStandard = 1

// module holds the state the three tools share.
type module struct {
	// sessionsRoot confines the transcript tools. Empty means the Claude Code
	// projects directory.
	sessionsRoot string
	// checklist returns the project's verification commands.
	checklist func() ([]string, error)
}

// Tools returns the agent-postmortem tool registrations. The verification
// checklist is derived from the languages recorded in projectRoot's qsdev
// answers file.
func Tools(projectRoot string) []spi.ToolRegistration {
	return newModule("", func() ([]string, error) { return verificationCommands(projectRoot) }).registrations()
}

func newModule(sessionsRoot string, checklist func() ([]string, error)) *module {
	return &module{sessionsRoot: sessionsRoot, checklist: checklist}
}

func (m *module) registrations() []spi.ToolRegistration {
	return []spi.ToolRegistration{
		{
			Name:        "analyze_session",
			Description: "Parse a Claude Code session transcript (.jsonl under the Claude projects directory) and return a structured analysis of its tool calls, failures and recoveries.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_path": map[string]any{
						"type":        "string",
						"description": "Path to the session .jsonl file (must be under the Claude projects directory)",
					},
				},
				"required": []any{"session_path"},
			},
			Category:    middleware.CategoryDiagnostics,
			Tier:        tierStandard,
			Annotations: spi.ReadOnlyAnnotations(false),
			Handler:     m.analyzeSession,
		},
		{
			Name:        "list_failure_patterns",
			Description: "Walk a directory of Claude Code session transcripts, parse each, and aggregate failure patterns across sessions.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"sessions_dir": map[string]any{
						"type":        "string",
						"description": "Directory to search for .jsonl session files (defaults to the Claude projects directory; must be inside it)",
					},
				},
			},
			Category:    middleware.CategoryDiagnostics,
			Tier:        tierStandard,
			Annotations: spi.ReadOnlyAnnotations(false),
			Handler:     m.listFailurePatterns,
		},
		{
			Name:        "generate_verification_checklist",
			Description: "Generate the project's verification checklist: the build, test and lint commands of the languages qsdev manages for it.",
			InputSchema: toolutil.EmptyObjectSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierStandard,
			Annotations: spi.ReadOnlyAnnotations(false),
			Handler:     m.generateChecklist,
		},
	}
}

func (m *module) analyzeSession(_ context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	sessionPath, _ := toolutil.StringArg(req.Arguments, "session_path")
	if sessionPath == "" {
		return toolutil.ErrorResult("missing required parameter: session_path", nil), nil
	}
	if filepath.Ext(sessionPath) != sessionExt {
		return toolutil.Denied("session_path must name a "+sessionExt+" session transcript", map[string]any{"session_path": sessionPath}), nil
	}
	resolved, denied := m.confine(sessionPath, "session_path")
	if denied != nil {
		return denied, nil
	}

	analysis, err := postmortem.ParseSessionJSONL(resolved)
	if err != nil {
		return toolutil.ErrorResult(err.Error(), nil), nil
	}
	return toolutil.Result(toolutil.MarshalText(analysis, "session analysis"), analysis), nil
}

func (m *module) listFailurePatterns(_ context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	sessionsDir := toolutil.StringArgOr(req.Arguments, "sessions_dir", "")
	if sessionsDir == "" {
		root, err := m.root()
		if err != nil {
			return toolutil.NotConfigured(err.Error(), nil), nil
		}
		sessionsDir = root
	}
	resolved, denied := m.confine(sessionsDir, "sessions_dir")
	if denied != nil {
		return denied, nil
	}

	scan, err := postmortem.FindSessionFiles(resolved, maxSessionFiles)
	if err != nil {
		return toolutil.ErrorResult(err.Error(), nil), nil
	}

	var analyses []*postmortem.SessionAnalysis
	skipped := scan.Unreadable
	for _, path := range scan.Paths {
		a, parseErr := postmortem.ParseSessionJSONL(path)
		if parseErr != nil {
			skipped++
			continue
		}
		analyses = append(analyses, a)
	}

	report := postmortem.AggregateFailures(analyses)
	report.SkippedFiles = skipped
	report.Truncated = scan.Truncated
	return toolutil.Result(toolutil.MarshalText(report, "failure report"), report), nil
}

func (m *module) generateChecklist(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
	if m.checklist == nil {
		return toolutil.NotConfigured("verification checklist is not available: no checklist source is configured", nil), nil
	}
	cmds, err := m.checklist()
	if err != nil {
		// Never substitute another ecosystem's commands: a Go checklist for a
		// Node project would be presented as project-specific and "pass".
		return toolutil.NotConfigured(fmt.Sprintf("generating verification checklist: %v", err), nil), nil
	}
	if cmds == nil {
		cmds = []string{}
	}
	return toolutil.Result(toolutil.MarshalText(cmds, "[]"), map[string]any{"commands": cmds}), nil
}

// verificationCommands returns the verification commands of the languages in
// projectRoot's primary qsdev answers file. A project without one has no
// recorded languages, which is reported rather than answered with an empty
// (vacuously passing) checklist.
func verificationCommands(projectRoot string) ([]string, error) {
	a, err := answers.RequirePrimary(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("loading qsdev answers: %w", err)
	}
	return ecosystem.LanguageVerificationCommands(a.Languages, ecosystem.DefaultRegistry()), nil
}

// root returns the directory the transcript tools are confined to.
func (m *module) root() (string, error) {
	if m.sessionsRoot != "" {
		return m.sessionsRoot, nil
	}
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "projects"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating the Claude projects directory: %w", err)
	}
	return filepath.Join(home, ".claude", "projects"), nil
}

// confine resolves path (following symlinks; a relative path is taken relative
// to the sessions root) and returns it when it lies inside the sessions root.
// Otherwise it returns the result to send back: a denial for a path outside the
// root, so an agent-supplied path cannot direct the tools to read or walk
// arbitrary parts of the filesystem.
func (m *module) confine(path, arg string) (string, *spi.ToolResult) {
	root, err := m.root()
	if err != nil {
		return "", toolutil.NotConfigured(err.Error(), nil)
	}
	realRoot, err := resolvePath(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", toolutil.NotConfigured("the Claude sessions directory "+root+" does not exist", nil)
		}
		return "", toolutil.ErrorResult(fmt.Sprintf("resolving sessions root %s: %v", root, err), nil)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	realPath, err := resolvePath(path)
	if err != nil {
		return "", toolutil.ErrorResult(fmt.Sprintf("resolving %s: %v", arg, err), map[string]any{arg: path})
	}
	rel, err := filepath.Rel(realRoot, realPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", toolutil.Denied(fmt.Sprintf("%s %s is outside the Claude sessions directory %s", arg, path, root), map[string]any{arg: path})
	}
	return realPath, nil
}

// resolvePath returns the absolute, symlink-free form of path.
func resolvePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("making %s absolute: %w", path, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolving symlinks in %s: %w", abs, err)
	}
	return resolved, nil
}
