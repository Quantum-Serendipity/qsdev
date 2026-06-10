package hardening

import "fmt"

type TrustLevel string

const (
	TrustUntrusted TrustLevel = "untrusted"
	TrustModerate  TrustLevel = "moderate"
	TrustTrusted   TrustLevel = "trusted"
)

func Frame(input, serverName string, tier int, source string) string {
	trust := tierToTrustLevel(tier)
	return fmt.Sprintf(
		"<qsdev:data server=%q tier=\"tier-%d\" source=%q trust=%q>\n%s\n</qsdev:data>",
		serverName, tier, source, trust, input,
	)
}

func tierToTrustLevel(tier int) TrustLevel {
	switch tier {
	case 1:
		return TrustTrusted
	case 2:
		return TrustModerate
	default:
		return TrustUntrusted
	}
}
