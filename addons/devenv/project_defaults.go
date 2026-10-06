package devenv

import (
	"fmt"
	"io"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/termutil"
)

// PrintProjectDefaults names the project defaults file the catalog applies
// (the layer of the root the runtime set for this command), if any, and the
// pre-commit hooks it adds, so the plan shows what the committed file will
// install. A file the trust rule refuses is not named, and a file that does
// not load lists no hooks: loading the catalog reports either. The file is
// repository content, so every value it supplies is shown through
// termutil.Safe: a hook entry cannot use a carriage return or an escape
// sequence to rewrite or hide its own line.
//
// Every command that generates devenv.nix from the catalog (init and update,
// devenv init, update, add and remove, enable and disable) calls it once,
// before it plans any file.
func PrintProjectDefaults(w io.Writer) {
	p, err := catalog.ProjectConfigFile(catalog.ProjectRoot())
	if err != nil || p == "" {
		return
	}
	_, _ = fmt.Fprintf(w, "Project defaults: %s\n", termutil.Safe(p))
	cat, err := catalog.Default()
	if err != nil {
		return
	}
	for _, h := range cat.ProjectOverlayHooks() {
		line := fmt.Sprintf("  adds pre-commit hook %s (%s)", termutil.Safe(h.ID), termutil.Safe(h.Section))
		if h.Entry != "" {
			line += ": " + termutil.Safe(h.Entry)
		}
		_, _ = fmt.Fprintln(w, line)
	}
}
