package devinit

import "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"

// ecosystems returns the ecosystem module registry init, join, update and
// check generate from and validate against. It is devinit's one reference to
// the process-wide registry.
func ecosystems() *ecosystem.Registry {
	return ecosystem.DefaultRegistry()
}

// declaredSettings returns the language settings the ecosystem modules
// declare: the languages[].extras keys .qsdev.yaml may commit.
func declaredSettings() ecosystem.DeclaredSettings {
	return ecosystems().DeclaredSettings()
}
