// Package aiframework defines the framework-agnostic contracts qsdev uses to
// detect, configure and secure AI coding frameworks (Claude Code, Codex,
// Cursor and others). Each framework supplies an adapter that implements the
// interfaces below; package contracttest holds the shared conformance suite.
//
// Stable interfaces with a production implementation in the Claude Code
// reference adapter (pkg/aiframework/adapters/claudecode):
//
//   - DetectionAdapter: finds a framework's markers in a project.
//   - ConfigRenderer: renders a PolicyInput into framework config files.
//   - ToolAdapter: translates permission, ignore and credential policy.
//
// Interfaces with no implementation yet (no adapter satisfies them, so a
// caller cannot mistake a stub's success for real behaviour):
//
//   - HookDeployer: deploys lifecycle hooks.
//   - RegistryClient: generates MCP server configuration.
//   - MetricsProvider: reports usage and budget metrics.
//
// # Experimental
//
// StateBackend and its chronicle and task types (ChronicleEntry,
// ChronicleVerb, TaskStatus, TaskOutcome, TaskSpec, TaskInfo, TaskFilter) are
// placeholders for planned multi-agent coordination. No backend exists.
// TaskStatus has no terminal states, so a completed task's outcome cannot yet
// be represented in TaskInfo.Status, and no transition rules are defined. Do
// not build on this API; it will change when the first real backend lands.
package aiframework
