package claudecode

import "github.com/Quantum-Serendipity/qsdev/pkg/denyutil"

func matchesDenyRule(denyRule, operation string) bool {
	return denyutil.MatchesDenyRule(denyRule, operation)
}

func parseToolPattern(pattern string) (string, string) {
	return denyutil.ParseToolPattern(pattern)
}
