package risk

import "time"

type Ecosystem string

const (
	EcosystemNpm      Ecosystem = "npm"
	EcosystemPyPI     Ecosystem = "pypi"
	EcosystemGo       Ecosystem = "go"
	EcosystemCargo    Ecosystem = "cargo"
	EcosystemNuGet    Ecosystem = "nuget"
	EcosystemRubyGems Ecosystem = "rubygems"
	EcosystemComposer Ecosystem = "composer"
)

type ProbeStatus string

const (
	ProbePass            ProbeStatus = "pass"
	ProbeFail            ProbeStatus = "fail"
	ProbeDataUnavailable ProbeStatus = "data_unavailable"
)

type RiskGrade string

const (
	GradeA RiskGrade = "A"
	GradeB RiskGrade = "B"
	GradeC RiskGrade = "C"
	GradeD RiskGrade = "D"
	GradeF RiskGrade = "F"
)

type PackageInfo struct {
	Name                    string
	Version                 string
	Ecosystem               Ecosystem
	PublishedAt             *time.Time
	FirstPublishedAt        *time.Time
	IsPreRelease            bool
	MaintainerCount         int
	HasInstallScripts       bool
	InstallScriptsBlocked   bool
	HasBinaries             bool
	CVECritical             int
	CVEHigh                 int
	CVEMedium               int
	CVELow                  int
	KEVListed               bool
	MalwareDetected         bool
	FixAvailable            bool
	EPSSMax                 float64
	Reachable               *bool
	DownloadCount           int64
	DependentCount          int
	HasChecksumVerification bool
	SLSALevel               int
	HasSignature            bool
	IsDirect                bool
	DepthFromRoot           int
}

type ProbeResult struct {
	ProbeID  string
	Category string
	Status   ProbeStatus
	RawValue float64
	Weight   float64
	Score    float64
}

type CategoryScore struct {
	Name     string
	Weight   float64
	RawScore float64
	Probes   []ProbeResult
}

type PackageScore struct {
	PackageName    string
	PackageVersion string
	Ecosystem      Ecosystem
	Score          int
	Grade          RiskGrade
	Categories     []CategoryScore
	CeilingApplied string
	Probes         []ProbeResult
	DataFreshness  time.Time
}

type DependencyHealth struct {
	TotalPackages     int
	GradeDistribution map[RiskGrade]int
	CriticalFindings  []PackageScore
	AggregateScore    int
	AggregateGrade    RiskGrade
}
