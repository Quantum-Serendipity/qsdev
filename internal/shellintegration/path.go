// Package shellintegration provides shell completion installation and
// shell-name normalization for the qsdev binary.
package shellintegration

import "strings"

func normalizeShellName(shell string) string {
	name := shell
	if i := strings.LastIndexAny(shell, `/\`); i >= 0 {
		name = shell[i+1:]
	}
	name = strings.TrimSuffix(name, ".exe")
	return strings.ToLower(name)
}
