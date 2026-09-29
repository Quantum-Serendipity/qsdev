package claudecode

import "embed"

// templateFS holds the shipped templates. The pattern deliberately omits the
// "all:" prefix so dot- and underscore-prefixed entries (.gitkeep, Python
// __pycache__ bytecode written by the hook tests, stray local state) are never
// embedded: they are not templates, and embedding them would make the binary
// and its template-version hash depend on the state of the build tree.
//
//go:embed templates
var templateFS embed.FS
