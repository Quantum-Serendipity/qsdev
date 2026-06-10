package trust

import "fmt"

type TrustTier int

const (
	Tier1Local      TrustTier = 1
	Tier2Enterprise TrustTier = 2
	Tier3Fallback   TrustTier = 3
)

var trustTierNames = map[TrustTier]string{
	Tier1Local:      "tier-1-local",
	Tier2Enterprise: "tier-2-enterprise",
	Tier3Fallback:   "tier-3-fallback",
}

func (t TrustTier) String() string {
	if name, ok := trustTierNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TrustTier(%d)", int(t))
}

type McpServerInfo struct {
	Name                    string
	Command                 string
	Args                    []string
	Env                     map[string]string
	Transport               string
	IsLocalBinary           bool
	ServesCommunityCContent bool
	HasContentSigning       bool
	VerifiedInstallSource   bool
	PinnedVersion           bool
	ControlledUpdates       bool
	OfflineCapable          bool
	HasKnownVulnerabilities bool
	HasUserAttestation      bool
}

type TrustScore struct {
	ServerName     string
	Score          int
	Tier           TrustTier
	Categories     []CategoryScore
	CeilingApplied string
	Probes         []ProbeResult
}

type ProbeResult struct {
	ProbeID  string
	Category string
	Pass     bool
	Weight   float64
}

type CategoryScore struct {
	Name   string
	Weight float64
	Score  float64
	Probes []ProbeResult
}

type McpTrustPosture struct {
	Tier1Count             int
	Tier2Count             int
	Tier3Count             int
	ConfusedDeputyActive   bool
	ProjectedDenyRuleCount int
}

type DenyRule struct {
	Pattern string
	Type    string
}

type TrustConfig struct {
	Servers map[string]TrustServerEntry `yaml:"servers"`
}

type TrustServerEntry struct {
	Tier           TrustTier `yaml:"tier"`
	Score          int       `yaml:"score"`
	ManualOverride bool      `yaml:"manual_override,omitempty"`
}
