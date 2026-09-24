package risk

import (
	"testing"
	"time"
)

// enrichedInfo returns an aged, checksum-verified package whose vulnerability
// lookup completed clean; callers layer vulnerability data on top.
func enrichedInfo(t *testing.T) PackageInfo {
	t.Helper()
	aged := time.Now().Add(-400 * 24 * time.Hour)
	return PackageInfo{
		Name:                    "pkg",
		Version:                 "1.0.0",
		Ecosystem:               EcosystemNpm,
		FirstPublishedAt:        &aged,
		PublishedAt:             &aged,
		VulnDataAvailable:       true,
		HasChecksumVerification: true,
	}
}

// TestScorePackage_LowerSeverityVulnDataCounts is the regression guard for the
// medium/low CVE, EPSS and fix-available probes being stubs: a package with
// hundreds of medium CVEs and near-certain exploitation must not grade A.
func TestScorePackage_LowerSeverityVulnDataCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(*PackageInfo)
		wantGrade RiskGrade
		notGrade  RiskGrade
	}{
		{"clean package keeps grade A", func(*PackageInfo) {}, GradeA, ""},
		{"mass medium/low CVEs with high EPSS", func(p *PackageInfo) {
			p.CVEMedium, p.CVELow, p.EPSSMax = 500, 500, 0.99
		}, "", GradeA},
		{"medium CVEs without a fix", func(p *PackageInfo) {
			p.CVEMedium = 10
		}, "", GradeA},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			info := enrichedInfo(t)
			tt.mutate(&info)
			got := ScorePackage(&info).Grade
			if tt.wantGrade != "" && got != tt.wantGrade {
				t.Errorf("grade = %s, want %s", got, tt.wantGrade)
			}
			if tt.notGrade != "" && got == tt.notGrade {
				t.Errorf("grade = %s, want anything but %s", got, tt.notGrade)
			}
		})
	}
}

func TestVulnerabilityProbes(t *testing.T) {
	t.Parallel()

	reachable, unreachable := true, false

	tests := []struct {
		name       string
		probe      ProbeFunc
		info       PackageInfo
		wantStatus ProbeStatus
		wantScore  float64
	}{
		{"cve-medium clean", probeCVEMedium, PackageInfo{VulnDataAvailable: true}, ProbePass, 100},
		{"cve-medium some", probeCVEMedium, PackageInfo{VulnDataAvailable: true, CVEMedium: 2}, ProbePass, 70},
		{"cve-medium many", probeCVEMedium, PackageInfo{VulnDataAvailable: true, CVEMedium: 500}, ProbeFail, 0},
		{"cve-medium unenriched", probeCVEMedium, PackageInfo{CVEMedium: 500}, ProbeDataUnavailable, 0},
		{"cve-low some", probeCVELow, PackageInfo{VulnDataAvailable: true, CVELow: 4}, ProbePass, 80},
		{"cve-low many", probeCVELow, PackageInfo{VulnDataAvailable: true, CVELow: 500}, ProbeFail, 0},
		{"epss negligible", probeEPSSMax, PackageInfo{VulnDataAvailable: true, EPSSMax: 0.001}, ProbePass, 100},
		{"epss moderate", probeEPSSMax, PackageInfo{VulnDataAvailable: true, EPSSMax: 0.05}, ProbePass, 70},
		{"epss high", probeEPSSMax, PackageInfo{VulnDataAvailable: true, EPSSMax: 0.2}, ProbeFail, 30},
		{"epss near-certain", probeEPSSMax, PackageInfo{VulnDataAvailable: true, EPSSMax: 0.99}, ProbeFail, 0},
		{"fix n/a when clean", probeFixAvailable, PackageInfo{VulnDataAvailable: true}, ProbePass, 100},
		{"fix available", probeFixAvailable, PackageInfo{VulnDataAvailable: true, CVEMedium: 1, FixAvailable: true}, ProbePass, 60},
		{"no fix", probeFixAvailable, PackageInfo{VulnDataAvailable: true, CVEHigh: 1}, ProbeFail, 20},
		{"reachability unknown", probeReachable, PackageInfo{VulnDataAvailable: true, CVEHigh: 1}, ProbeDataUnavailable, 0},
		{"reachable vuln", probeReachable, PackageInfo{VulnDataAvailable: true, CVEHigh: 1, Reachable: &reachable}, ProbeFail, 0},
		{"unreachable vuln", probeReachable, PackageInfo{VulnDataAvailable: true, CVEHigh: 1, Reachable: &unreachable}, ProbePass, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.probe(&tt.info)
			if got.Status != tt.wantStatus || got.Score != tt.wantScore {
				t.Errorf("probe = status %s score %v, want %s %v", got.Status, got.Score, tt.wantStatus, tt.wantScore)
			}
		})
	}
}
