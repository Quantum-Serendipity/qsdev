package vulnscan

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// osvStub builds an httptest server that answers the OSV /v1/querybatch and
// /v1/vulns/{id} endpoints. vulnsByIndex maps a query index to the vuln ids to
// report for it; severities maps a vuln id to the database_specific.severity
// label returned for its detail record.
func osvStub(t *testing.T, vulnsByIndex map[int][]string, severities map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/v1/querybatch"):
			var req osvBatchRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			var resp osvBatchResponse
			resp.Results = make([]struct {
				Vulns []struct {
					ID string `json:"id"`
				} `json:"vulns"`
			}, len(req.Queries))
			for i := range req.Queries {
				for _, id := range vulnsByIndex[i] {
					resp.Results[i].Vulns = append(resp.Results[i].Vulns, struct {
						ID string `json:"id"`
					}{ID: id})
				}
			}
			_ = json.NewEncoder(w).Encode(resp)
		case strings.Contains(r.URL.Path, "/v1/vulns/"):
			id := strings.TrimPrefix(r.URL.Path, "/v1/vulns/")
			v := osvVuln{ID: id}
			v.DatabaseSpecific.Severity = severities[id]
			_ = json.NewEncoder(w).Encode(v)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeLock(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

func TestScanFile_CriticalAdvisory(t *testing.T) {
	dir := t.TempDir()
	lock := writeLock(t, dir, "requirements.txt", "requests==2.19.0\n")

	srv := osvStub(t,
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

	srv := osvStub(t,
		map[int][]string{0: {"A-high"}, 1: {"B-moderate", "C-unknown"}},
		map[string]string{"A-high": "HIGH", "B-moderate": "MODERATE"}, // C-unknown has no severity
	)
	s := &Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()}

	res, err := s.ScanFile(context.Background(), lock)
	if err != nil {
		t.Fatalf("ScanFile: %v", err)
	}
	if res.Counts.High != 1 || res.Counts.Moderate != 1 || res.Counts.Info != 1 {
		t.Errorf("counts = %+v, want High=1 Moderate=1 Info=1", res.Counts)
	}
	if res.Counts.Critical != 0 {
		t.Errorf("Counts.Critical = %d, want 0", res.Counts.Critical)
	}
}

func TestScanFile_CleanProject(t *testing.T) {
	dir := t.TempDir()
	lock := writeLock(t, dir, "requirements.txt", "requests==2.31.0\n")

	srv := osvStub(t, map[int][]string{}, map[string]string{}) // no vulns for any query
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

	srv := osvStub(t,
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
