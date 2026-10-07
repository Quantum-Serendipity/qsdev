package catalog

import "testing"

// BenchmarkCatalogLoad times loading the embedded catalog with no project
// or organization overlay: the parse and validation every command that
// reads the catalog pays once per process. Run it with `make bench`.
func BenchmarkCatalogLoad(b *testing.B) {
	for b.Loop() {
		if _, err := Load(); err != nil {
			b.Fatal(err)
		}
	}
}
