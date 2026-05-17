// Package ecosystem defines the SASTModule optional interface for ecosystem
// modules that provide static analysis rule sets.
package ecosystem

// SASTModule is an optional interface that ecosystem modules can implement
// to declare which Semgrep rule sets are relevant for their language/platform.
// Modules that implement this interface contribute rule sets to the generated
// .semgrep.yml configuration file.
type SASTModule interface {
	SemgrepRuleSets() []string
}
