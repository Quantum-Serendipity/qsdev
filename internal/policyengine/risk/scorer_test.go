package risk

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScorePackage(t *testing.T) {
	t.Parallel()

	twoYearsAgo := time.Now().Add(-2 * 365 * 24 * time.Hour)
	sixMonthsAgo := time.Now().Add(-180 * 24 * time.Hour)

	tests := []struct {
		name           string
		info           PackageInfo
		wantMaxScore   int
		wantMinScore   int
		wantGrade      RiskGrade
		wantCeiling    string
		checkGradeOnly bool
	}{
		{
			name: "kev-listed package capped at 5",
			info: PackageInfo{
				Name:                    "vuln-pkg",
				Version:                 "1.0.0",
				Ecosystem:               EcosystemNpm,
				FirstPublishedAt:        &twoYearsAgo,
				PublishedAt:             &sixMonthsAgo,
				KEVListed:               true,
				HasChecksumVerification: true,
			},
			wantMaxScore: 5,
			wantMinScore: 0,
			wantGrade:    GradeF,
			wantCeiling:  "kev",
		},
		{
			name: "critical CVE with fix capped at 15",
			info: PackageInfo{
				Name:                    "crit-pkg",
				Version:                 "2.0.0",
				Ecosystem:               EcosystemNpm,
				FirstPublishedAt:        &twoYearsAgo,
				PublishedAt:             &sixMonthsAgo,
				CVECritical:             1,
				FixAvailable:            true,
				HasChecksumVerification: true,
			},
			wantMaxScore: 15,
			wantMinScore: 0,
			wantGrade:    GradeF,
			wantCeiling:  "critical-cve-fix-available",
		},
		{
			name: "healthy package scores A",
			info: PackageInfo{
				Name:                    "healthy-pkg",
				Version:                 "3.0.0",
				Ecosystem:               EcosystemNpm,
				FirstPublishedAt:        &twoYearsAgo,
				PublishedAt:             &sixMonthsAgo,
				HasChecksumVerification: true,
				IsDirect:                true,
			},
			wantMinScore:   90,
			wantMaxScore:   100,
			wantGrade:      GradeA,
			wantCeiling:    "",
			checkGradeOnly: false,
		},
		{
			name: "malware detected scores 0",
			info: PackageInfo{
				Name:                    "malware-pkg",
				Version:                 "0.0.1",
				Ecosystem:               EcosystemNpm,
				FirstPublishedAt:        &twoYearsAgo,
				PublishedAt:             &sixMonthsAgo,
				MalwareDetected:         true,
				HasChecksumVerification: true,
			},
			wantMaxScore: 0,
			wantMinScore: 0,
			wantGrade:    GradeF,
			wantCeiling:  "malware",
		},
		{
			name: "unblocked install scripts capped at 40",
			info: PackageInfo{
				Name:                    "scripts-pkg",
				Version:                 "1.0.0",
				Ecosystem:               EcosystemNpm,
				FirstPublishedAt:        &twoYearsAgo,
				PublishedAt:             &sixMonthsAgo,
				HasInstallScripts:       true,
				InstallScriptsBlocked:   false,
				HasChecksumVerification: true,
			},
			wantMaxScore: 40,
			wantMinScore: 0,
			wantGrade:    GradeF,
			wantCeiling:  "unblocked-install-scripts",
		},
		{
			name: "all data unavailable returns grade F with score 0",
			info: PackageInfo{
				Name:      "unknown-pkg",
				Version:   "0.1.0",
				Ecosystem: EcosystemNpm,
			},
			wantMinScore:   0,
			wantMaxScore:   100,
			checkGradeOnly: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ScorePackage(&tt.info)

			if result.Score < tt.wantMinScore {
				t.Errorf("score %d below minimum %d", result.Score, tt.wantMinScore)
			}
			if result.Score > tt.wantMaxScore {
				t.Errorf("score %d above maximum %d", result.Score, tt.wantMaxScore)
			}
			if tt.wantGrade != "" && result.Grade != tt.wantGrade {
				t.Errorf("grade = %s, want %s", result.Grade, tt.wantGrade)
			}
			if tt.wantCeiling != "" && result.CeilingApplied != tt.wantCeiling {
				t.Errorf("ceiling = %q, want %q", result.CeilingApplied, tt.wantCeiling)
			}
		})
	}
}

func TestScoreAll(t *testing.T) {
	t.Parallel()

	twoYearsAgo := time.Now().Add(-2 * 365 * 24 * time.Hour)
	sixMonthsAgo := time.Now().Add(-180 * 24 * time.Hour)

	packages := []PackageInfo{
		{
			Name:                    "good-pkg",
			Version:                 "1.0.0",
			Ecosystem:               EcosystemNpm,
			FirstPublishedAt:        &twoYearsAgo,
			PublishedAt:             &sixMonthsAgo,
			HasChecksumVerification: true,
			IsDirect:                true,
		},
		{
			Name:            "bad-pkg",
			Version:         "0.0.1",
			Ecosystem:       EcosystemNpm,
			MalwareDetected: true,
			IsDirect:        false,
		},
		{
			Name:                    "ok-pkg",
			Version:                 "2.0.0",
			Ecosystem:               EcosystemNpm,
			FirstPublishedAt:        &twoYearsAgo,
			PublishedAt:             &sixMonthsAgo,
			HasChecksumVerification: true,
			IsDirect:                true,
		},
	}

	health := ScoreAll(packages)

	if health.TotalPackages != 3 {
		t.Errorf("total packages = %d, want 3", health.TotalPackages)
	}

	if health.GradeDistribution[GradeF] < 1 {
		t.Errorf("expected at least 1 F-grade package, got %d", health.GradeDistribution[GradeF])
	}

	if len(health.CriticalFindings) < 1 {
		t.Errorf("expected at least 1 critical finding, got %d", len(health.CriticalFindings))
	}

	aCount := health.GradeDistribution[GradeA]
	if aCount < 1 {
		t.Errorf("expected at least 1 A-grade package, got %d", aCount)
	}
}

func TestScoreAllEmpty(t *testing.T) {
	t.Parallel()

	health := ScoreAll(nil)
	if health.TotalPackages != 0 {
		t.Errorf("total packages = %d, want 0", health.TotalPackages)
	}
	if health.AggregateScore != 0 {
		t.Errorf("aggregate score = %d, want 0", health.AggregateScore)
	}
}

func TestCacheManager(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cache := NewCacheManager(dir, 1*time.Hour)

	got, err := cache.Get("npm", "test-pkg", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error on cache miss: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for cache miss")
	}

	score := &PackageScore{
		PackageName:    "test-pkg",
		PackageVersion: "1.0.0",
		Ecosystem:      EcosystemNpm,
		Score:          85,
		Grade:          GradeB,
	}

	if err := cache.Put(score); err != nil {
		t.Fatalf("unexpected error on cache put: %v", err)
	}

	got, err = cache.Get("npm", "test-pkg", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error on cache hit: %v", err)
	}
	if got == nil {
		t.Fatal("expected cached score, got nil")
		return
	}
	if got.Score != 85 {
		t.Errorf("cached score = %d, want 85", got.Score)
	}
	if got.Grade != GradeB {
		t.Errorf("cached grade = %s, want B", got.Grade)
	}
}

func TestCacheManagerExpiry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cache := NewCacheManager(dir, 1*time.Millisecond)

	score := &PackageScore{
		PackageName:    "expire-pkg",
		PackageVersion: "1.0.0",
		Ecosystem:      EcosystemNpm,
		Score:          50,
		Grade:          GradeD,
	}

	if err := cache.Put(score); err != nil {
		t.Fatalf("unexpected error on cache put: %v", err)
	}

	// Set mtime to the past to simulate expiry without sleeping.
	path := filepath.Join(dir, cache.cachePath("npm", "expire-pkg", "1.0.0"))
	// cachePath returns the full path already, so re-derive it.
	key := cache.cachePath("npm", "expire-pkg", "1.0.0")
	pastTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(key, pastTime, pastTime); err != nil {
		t.Fatalf("setting mtime: %v", err)
	}
	_ = path

	got, err := cache.Get("npm", "expire-pkg", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for expired cache entry")
	}
}

func TestApplyCeilings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		rawScore    int
		info        PackageInfo
		wantScore   int
		wantCeiling string
	}{
		{
			name:        "no ceiling",
			rawScore:    85,
			info:        PackageInfo{},
			wantScore:   85,
			wantCeiling: "",
		},
		{
			name:        "malware ceiling",
			rawScore:    85,
			info:        PackageInfo{MalwareDetected: true},
			wantScore:   0,
			wantCeiling: "malware",
		},
		{
			name:        "kev ceiling",
			rawScore:    85,
			info:        PackageInfo{KEVListed: true},
			wantScore:   5,
			wantCeiling: "kev",
		},
		{
			name:        "critical cve with fix",
			rawScore:    85,
			info:        PackageInfo{CVECritical: 1, FixAvailable: true},
			wantScore:   15,
			wantCeiling: "critical-cve-fix-available",
		},
		{
			name:        "critical cve no fix",
			rawScore:    85,
			info:        PackageInfo{CVECritical: 2},
			wantScore:   25,
			wantCeiling: "critical-cve-no-fix",
		},
		{
			name:        "unblocked install scripts",
			rawScore:    85,
			info:        PackageInfo{HasInstallScripts: true},
			wantScore:   40,
			wantCeiling: "unblocked-install-scripts",
		},
		{
			name:        "malware takes priority over kev",
			rawScore:    85,
			info:        PackageInfo{MalwareDetected: true, KEVListed: true},
			wantScore:   0,
			wantCeiling: "malware",
		},
		{
			name:        "score already below ceiling is unchanged",
			rawScore:    3,
			info:        PackageInfo{KEVListed: true},
			wantScore:   3,
			wantCeiling: "kev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			score, ceiling := ApplyCeilings(tt.rawScore, &tt.info)
			if score != tt.wantScore {
				t.Errorf("score = %d, want %d", score, tt.wantScore)
			}
			if ceiling != tt.wantCeiling {
				t.Errorf("ceiling = %q, want %q", ceiling, tt.wantCeiling)
			}
		})
	}
}

func TestProbeEcosystemFiltering(t *testing.T) {
	t.Parallel()

	goInfo := &PackageInfo{
		Name:      "go-pkg",
		Version:   "1.0.0",
		Ecosystem: EcosystemGo,
	}
	npmInfo := &PackageInfo{
		Name:      "npm-pkg",
		Version:   "1.0.0",
		Ecosystem: EcosystemNpm,
	}

	goResult := ScorePackage(goInfo)
	npmResult := ScorePackage(npmInfo)

	goHasReachable := false
	for _, p := range goResult.Probes {
		if p.ProbeID == "reachable" {
			goHasReachable = true
			break
		}
	}
	if !goHasReachable {
		t.Error("Go package should have reachable probe")
	}

	npmHasReachable := false
	for _, p := range npmResult.Probes {
		if p.ProbeID == "reachable" {
			npmHasReachable = true
			break
		}
	}
	if npmHasReachable {
		t.Error("npm package should not have reachable probe")
	}

	// npm-provenance should appear for npm but not Go
	npmHasProvenance := false
	for _, p := range npmResult.Probes {
		if p.ProbeID == "npm-provenance" {
			npmHasProvenance = true
			break
		}
	}
	if !npmHasProvenance {
		t.Error("npm package should have npm-provenance probe")
	}

	goHasProvenance := false
	for _, p := range goResult.Probes {
		if p.ProbeID == "npm-provenance" {
			goHasProvenance = true
			break
		}
	}
	if goHasProvenance {
		t.Error("Go package should not have npm-provenance probe")
	}
}

func TestGradeFromScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		score int
		want  RiskGrade
	}{
		{100, GradeA},
		{90, GradeA},
		{89, GradeB},
		{80, GradeB},
		{79, GradeC},
		{70, GradeC},
		{69, GradeD},
		{50, GradeD},
		{49, GradeF},
		{0, GradeF},
	}

	for _, tt := range tests {
		t.Run(string(tt.want)+" boundary", func(t *testing.T) {
			t.Parallel()
			got := gradeFromScore(tt.score)
			if got != tt.want {
				t.Errorf("gradeFromScore(%d) = %s, want %s", tt.score, got, tt.want)
			}
		})
	}
}
