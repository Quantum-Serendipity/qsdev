package check

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorDim    = "\033[2m"
	colorBold   = "\033[1m"
)

// statusSymbol returns the display symbol for a check status.
// When useColor is true, ANSI color codes are applied.
func statusSymbol(status CheckStatus, useColor bool) string {
	if useColor {
		switch status {
		case StatusPass:
			return colorGreen + "✓" + colorReset
		case StatusFail:
			return colorRed + "✗" + colorReset
		case StatusWarn:
			return colorYellow + "⚠" + colorReset
		case StatusSkip:
			return colorDim + "-" + colorReset
		default:
			return "?"
		}
	}

	switch status {
	case StatusPass:
		return "[PASS]"
	case StatusFail:
		return "[FAIL]"
	case StatusWarn:
		return "[WARN]"
	case StatusSkip:
		return "[SKIP]"
	default:
		return "[????]"
	}
}
