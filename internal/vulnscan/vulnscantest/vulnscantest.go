// Package vulnscantest provides a parameterized in-process stub of the OSV.dev
// API for tests that exercise the vulnscan Scanner or its consumers (posture,
// the mcpserve security_scan tool) offline. It is a regular package rather than
// a _test file so tests in other packages can share it; import it from _test
// files only.
package vulnscantest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// batchRequest mirrors the subset of the OSV /v1/querybatch request the stub
// reads: only the number of queries matters, to size the response.
type batchRequest struct {
	Queries []json.RawMessage `json:"queries"`
}

// batchVuln, batchResult, and batchResponse mirror the /v1/querybatch response
// contract.
type batchVuln struct {
	ID string `json:"id"`
}

type batchResult struct {
	Vulns []batchVuln `json:"vulns"`
}

type batchResponse struct {
	Results []batchResult `json:"results"`
}

// vulnRecord mirrors the subset of a /v1/vulns/{id} record the scanner reads.
type vulnRecord struct {
	ID               string `json:"id"`
	DatabaseSpecific struct {
		Severity string `json:"severity"`
	} `json:"database_specific"`
}

// NewServer returns an httptest server answering the two OSV.dev endpoints the
// vulnscan Scanner uses. vulnsByIndex maps a /v1/querybatch query index to the
// vulnerability ids reported for that query; severities maps a vulnerability id
// to the database_specific.severity label its /v1/vulns/{id} record carries.
// Either map may be nil: a nil vulnsByIndex reports no vulnerabilities, and an
// id absent from severities yields a detail record with no severity label. The
// server is closed via t.Cleanup, so it outlives parallel subtests of t.
func NewServer(t testing.TB, vulnsByIndex map[int][]string, severities map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/v1/querybatch"):
			var req batchRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			resp := batchResponse{Results: make([]batchResult, len(req.Queries))}
			for i := range resp.Results {
				for _, id := range vulnsByIndex[i] {
					resp.Results[i].Vulns = append(resp.Results[i].Vulns, batchVuln{ID: id})
				}
			}
			_ = json.NewEncoder(w).Encode(resp)
		case strings.Contains(r.URL.Path, "/v1/vulns/"):
			id := strings.TrimPrefix(r.URL.Path, "/v1/vulns/")
			rec := vulnRecord{ID: id}
			rec.DatabaseSpecific.Severity = severities[id]
			_ = json.NewEncoder(w).Encode(rec)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}
