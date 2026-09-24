// Package ecosystem defines the SASTModule optional interface for ecosystem
// modules that provide static analysis rule sets.
package ecosystem

// SASTModule is an optional interface that ecosystem modules can implement
// to declare which Semgrep rule sets are relevant for their language/platform.
// The security-scan task passes each declared rule set to semgrep as a
// --config flag (see semgrepScanCommand).
type SASTModule interface {
	SemgrepRuleSets() []string
}
