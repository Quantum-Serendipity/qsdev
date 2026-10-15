package devinit

import "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"

// ecosystems returns the ecosystem module registry init, join, update and
// check generate from and validate against. It is devinit's one reference to
// the process-wide registry.
func ecosystems() *ecosystem.Registry {
	return ecosystem.DefaultRegistry()
}

// committableSettings returns the committable language settings of the
// ecosystem modules: the languages[].extras keys .qsdev.yaml may record.
func committableSettings() ecosystem.CommittableSettings {
	return ecosystems().CommittableSettings()
}
