package vulnscan

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/vulnscan/vulnscantest"
)

func writeLock(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

// TestNormalizeSeverity pins the single shared severity mapping used by both the
// CLI scan and the mcpserve security_scan tool (M11), so the same advisory can
// never report a different severity across entry points. Notably a GHSA
// "MODERATE" and a "MEDIUM" both map to "moderate", and an absent label maps to
// "unknown" (fail-closed), never silently to a low/hidden bucket.
func TestNormalizeSeverity(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"CRITICAL": "critical",
		"High":     "high",
		"MODERATE": "moderate",
		"medium":   "moderate",
		"LOW":      "low",
		"":         "unknown",
		"weird":    "info",
	}
	for label, want := range cases {
		if got := NormalizeSeverity(label); got != want {
			t.Errorf("NormalizeSeverity(%q) = %q, want %q", label, got, want)
		}
	}

	// SeverityAtOrAbove: unknown always qualifies; moderate meets a moderate
	// floor; low/info fall below it.
	if !SeverityAtOrAbove("unknown", "critical") {
		t.Error("unknown must qualify against any floor (fail-closed)")
	}
	if !SeverityAtOrAbove("moderate", "moderate") {
		t.Error("moderate must meet a moderate floor")
	}
	if SeverityAtOrAbove("low", "moderate") || SeverityAtOrAbove("info", "low") {
		t.Error("a severity below the floor must not qualify")
	}
}

func TestScanFile_CriticalAdvisory(t *testing.T) {
	dir := t.TempDir()
	lock := writeLock(t, dir, "requirements.txt", "requests==2.19.0\n")

	srv := vulnscantest.NewServer(t,
		map[int][]string{0: {"GHSA-critical-1"}},
		map[string]string{"GHSA-critical-1": "CRITICAL"},
	)
	s := &Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()}

	res, err := s.ScanFile(context.Background(), lock)
	if err != nil {
		t.Fatalf("ScanFile: %v", err)
	}
	if res == nil {
		t.Fatal("expected a result for a supported lock file")
	}
	if res.Dependencies != 1 {
		t.Errorf("Dependencies = %d, want 1", res.Dependencies)
	}
	if res.Counts.Critical != 1 {
		t.Errorf("Counts.Critical = %d, want 1", res.Counts.Critical)
	}
	if res.Counts.Total() != 1 {
		t.Errorf("Counts.Total = %d, want 1", res.Counts.Total())
	}
	if len(res.Vulnerabilities) != 1 || res.Vulnerabilities[0].Severity != "critical" {
		t.Errorf("vulnerabilities = %+v, want one critical", res.Vulnerabilities)
	}
	if res.Ecosystem != "PyPI" {
		t.Errorf("Ecosystem = %q, want PyPI", res.Ecosystem)
	}
}

func TestScanFile_MixedSeverities(t *testing.T) {
	dir := t.TempDir()
	lock := writeLock(t, dir, "requirements.txt", "requests==2.19.0\nflask==0.12\n")

	srv := vulnscantest.NewServer(t,
		map[int][]string{0: {"A-high"}, 1: {"B-moderate", "C-unknown"}},
		map[string]string{"A-high": "HIGH", "B-moderate": "MODERATE"}, // C-unknown has no severity
	)
	s := &Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()}

	res, err := s.ScanFile(context.Background(), lock)
	if err != nil {
		t.Fatalf("ScanFile: %v", err)
	}
	// C-unknown has no severity label; a missing/failed detail severity must be
	// counted as Unknown (fail-closed), NOT silently demoted to Info (M2).
	if res.Counts.High != 1 || res.Counts.Moderate != 1 || res.Counts.Unknown != 1 {
		t.Errorf("counts = %+v, want High=1 Moderate=1 Unknown=1", res.Counts)
	}
	if res.Counts.Info != 0 {
		t.Errorf("Counts.Info = %d, want 0 (unknown severity must not fold to Info)", res.Counts.Info)
	}
	if res.Counts.Critical != 0 {
		t.Errorf("Counts.Critical = %d, want 0", res.Counts.Critical)
	}
}

// TestScanFile_CVSSSeverityFallback proves the scanner derives severity from an
// advisory's top-level CVSS severity[] vector when database_specific.severity is
// absent — the shape Go advisories (GO-YYYY-NNNN) use. Without this fallback the
// label is empty, NormalizeSeverity yields "unknown", and the audit exit gate is
// pinned non-zero at every audit level for any Go project carrying a single
// advisory (#2).
func TestScanFile_CVSSSeverityFallback(t *testing.T) {
	cases := []struct {
		name         string
		vector       string
		wantSeverity string
	}{
		// AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H is CVSS 3.1 base 9.8.
		{"critical CVSS v3.1", "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", "critical"},
		// AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N is CVSS 3.1 base 5.3.
		{"moderate CVSS v3.1", "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", "moderate"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			lock := writeLock(t, dir, "requirements.txt", "requests==2.19.0\n")

			// The advisory carries NO database_specific.severity: its only severity
			// is the top-level CVSS vector.
			srv := vulnscantest.NewServerWithCVSS(t,
				map[int][]string{0: {"GO-2024-0001"}},
				nil,
				map[string]string{"GO-2024-0001": tc.vector},
			)
			s := &Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()}

			res, err := s.ScanFile(context.Background(), lock)
			if err != nil {
				t.Fatalf("ScanFile: %v", err)
			}
			if res.Counts.Unknown != 0 {
				t.Errorf("Counts.Unknown = %d, want 0 (CVSS vector must resolve severity, not fail-closed)", res.Counts.Unknown)
			}
			if len(res.Vulnerabilities) != 1 || res.Vulnerabilities[0].Severity != tc.wantSeverity {
				t.Fatalf("vulnerabilities = %+v, want one %q", res.Vulnerabilities, tc.wantSeverity)
			}
		})
	}
}

func TestScanFile_CleanProject(t *testing.T) {
	dir := t.TempDir()
	lock := writeLock(t, dir, "requirements.txt", "requests==2.31.0\n")

	srv := vulnscantest.NewServer(t, nil, nil) // no vulns for any query
	s := &Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()}

	res, err := s.ScanFile(context.Background(), lock)
	if err != nil {
		t.Fatalf("ScanFile: %v", err)
	}
	if res.Counts.Total() != 0 {
		t.Errorf("Counts.Total = %d, want 0", res.Counts.Total())
	}
}

func TestScanFile_UnsupportedLockFormat(t *testing.T) {
	dir := t.TempDir()
	lock := writeLock(t, dir, "yarn.lock", "# yarn lockfile v1\n")

	s := New() // never contacts the network for an unsupported format
	res, err := s.ScanFile(context.Background(), lock)
	if err != nil {
		t.Fatalf("ScanFile: %v", err)
	}
	if res != nil {
		t.Errorf("expected nil result for an unsupported lock format, got %+v", res)
	}
}

func TestScanProject_AutoDetects(t *testing.T) {
	dir := t.TempDir()
	writeLock(t, dir, "requirements.txt", "requests==2.19.0\n")

	srv := vulnscantest.NewServer(t,
		map[int][]string{0: {"GHSA-critical-1"}},
		map[string]string{"GHSA-critical-1": "CRITICAL"},
	)
	s := &Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()}

	res, err := s.ScanProject(context.Background(), dir)
	if err != nil {
		t.Fatalf("ScanProject: %v", err)
	}
	if res == nil || res.Counts.Critical != 1 {
		t.Fatalf("ScanProject result = %+v, want one critical", res)
	}
}

func TestScanProject_NoLockFile(t *testing.T) {
	s := New()
	res, err := s.ScanProject(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("ScanProject: %v", err)
	}
	if res != nil {
		t.Errorf("expected nil result when no lock file is present, got %+v", res)
	}
}

// TestFetchDetails_DeterministicTruncation proves that when more than
// maxVulnDetailFetches unique vulnerabilities are present, the subset whose
// details are fetched is deterministic (the lexicographically smallest ids) and
// stable across repeated runs — even with the fetches fanned out across the
// worker pool — so downstream severity filtering does not vary run-to-run.
func TestFetchDetails_DeterministicTruncation(t *testing.T) {
	t.Parallel()

	const total = maxVulnDetailFetches + 50
	oneQuery := make([]string, total)
	for i := range oneQuery {
		oneQuery[i] = fmt.Sprintf("VULN-%04d", i)
	}
	idsByQuery := [][]string{oneQuery}

	srv := vulnscantest.NewServer(t, nil, nil)
	s := &Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()}

	first := s.FetchDetails(context.Background(), idsByQuery)
	if len(first) != maxVulnDetailFetches {
		t.Fatalf("fetched %d details, want %d", len(first), maxVulnDetailFetches)
	}

	// The fetched subset must be exactly the lexicographically smallest ids.
	want := append([]string(nil), oneQuery...)
	sort.Strings(want)
	want = want[:maxVulnDetailFetches]
	for _, id := range want {
		if _, ok := first[id]; !ok {
			t.Fatalf("expected smallest id %q to be fetched", id)
		}
	}

	second := s.FetchDetails(context.Background(), idsByQuery)
	if len(second) != len(first) {
		t.Fatalf("second fetch size %d != first %d", len(second), len(first))
	}
	for id := range first {
		if _, ok := second[id]; !ok {
			t.Errorf("nondeterministic selection: id %q fetched first run but not second", id)
		}
	}
}
