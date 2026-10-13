package claudecode

import "github.com/Quantum-Serendipity/qsdev/pkg/denyutil"

func parseToolPattern(pattern string) (string, string) {
	return denyutil.ParseToolPattern(pattern)
}
