package sectools

// defaultScanExcludes contains directory and file patterns excluded from static
// analysis tools. Shared between Semgrep and OpenGrep to ensure consistent
// coverage boundaries.
var defaultScanExcludes = []string{
	// Build output
	"build/",
	"dist/",
	"out/",
	"result",
	"target/",

	// Caches
	".cache/",
	".pytest_cache/",
	"__pycache__/",

	// Coverage
	".coverage/",
	"coverage/",

	// Framework output
	".next/",
	".nuxt/",

	// Nix / devenv
	".devenv/",
	".direnv/",

	// Test fixtures
	"fixtures/",
	"testdata/",

	// Vendored / third-party dependencies
	"node_modules/",
	"third_party/",
	"vendor/",

	// Virtual environments
	".egg-info/",
	".tox/",
	".venv/",
	"venv/",
}
