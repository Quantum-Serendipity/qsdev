package catalog

// Catalog is a stub.
type Catalog struct{}

// MustDefault returns the singleton.
func MustDefault() *Catalog { return Default() }

// Default is called unqualified inside the owner package.
func Default() *Catalog { return &Catalog{} }
