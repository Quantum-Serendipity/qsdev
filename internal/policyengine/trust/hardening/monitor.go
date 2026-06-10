package hardening

type MonitorResult struct {
	Suspicious bool
	Patterns   []string
}

func CheckPostDocFetch(_ string, _ []string) MonitorResult {
	return MonitorResult{Suspicious: false}
}
