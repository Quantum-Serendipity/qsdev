package claudecode

import "embed"

// templateFS holds the shipped templates. The pattern deliberately omits the
// "all:" prefix so dot- and underscore-prefixed entries (.gitkeep, Python
// __pycache__ bytecode written by the hook tests, stray local state) are never
// embedded: they are not templates, and embedding them would make the binary
// and its template-version hash depend on the state of the build tree. The
// shared Python hook library is the one underscore-prefixed template, so it
// is named explicitly.
//
//go:embed templates templates/hooks/_qsdev_hooklib.py
var templateFS embed.FS
