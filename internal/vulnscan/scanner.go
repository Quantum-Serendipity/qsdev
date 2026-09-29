// Package vulnscan scans a project's pinned dependencies for known
// vulnerabilities by querying the OSV.dev database. It extracts dependency
// coordinates from lock files (go.sum, package-lock.json, Cargo.lock,
// poetry.lock, uv.lock, Pipfile.lock, requirements.txt), batches them to
// OSV.dev, and aggregates the advisories by severity.
//
// This package is the single lock-file parsing and OSV client implementation:
// internal/posture and the mcpserve security_scan tool both scan through the
// ScanFile entry point (locating the file with LockFileForEcosystem or
// DetectLockFile), so scan semantics such as batch chunking, result validation
// and the no-pinned-dependencies signal are defined once, here.
package vulnscan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// DefaultOSVBaseURL is the OSV.dev REST API root. It is the default value of
// Scanner.BaseURL; tests point BaseURL at a local httptest server instead.
const DefaultOSVBaseURL = "https://api.osv.dev"

// scanHTTPTimeout bounds the whole OSV interaction (batch query plus detail
// fetches) so a slow or unreachable OSV endpoint cannot stall the scan.
const scanHTTPTimeout = 20 * time.Second

// maxVulnDetailFetches caps how many unique vulnerability records the scanner
// fetches full details for, bounding fan-out on a heavily-vulnerable project.
const maxVulnDetailFetches = 200

// maxBatchQueries is the largest number of queries OSV.dev accepts in one
// /v1/querybatch request; larger batches are rejected with HTTP 400 ("too many
// queries"), so QueryBatch splits the dependency set into chunks of this size.
const maxBatchQueries = 1000

// maxPageRounds bounds how many follow-up requests QueryBatch makes for queries
// whose results OSV paginated (next_page_token). Exceeding it is an error rather
// than a silent truncation of the vulnerability list.
const maxPageRounds = 20

// maxBatchResponseBytes and maxVulnResponseBytes bound how much of an OSV
// response body is decoded, so a misbehaving endpoint or proxy cannot make the
// scanner buffer an unbounded body. An over-long body fails to decode and is
// reported as an error, never as a clean result.
const (
	maxBatchResponseBytes = 16 << 20
	maxVulnResponseBytes  = 4 << 20
)

// ErrNoPinnedDeps reports a lock file that parsed but yielded no pinned
// dependency coordinates (e.g. a requirements.txt of only loose specifiers such
// as "requests>=2"). Nothing was sent to OSV, so callers must treat the
// ecosystem as not scanned rather than as scanned clean.
var ErrNoPinnedDeps = errors.New("lock file has no pinned dependencies to scan")

// ErrLockParse reports a lock file that was found but could not be parsed
// (truncated, tampered with, or malformed). The scan did not run.
var ErrLockParse = errors.New("lock file could not be parsed")

// detailFetchConcurrency bounds the parallel /v1/vulns/{id} detail fetches so
// up to maxVulnDetailFetches records fit within the shared scan deadline
// (sequential fetches could not) without hammering the OSV endpoint.
const detailFetchConcurrency = 8

// SeverityCounts aggregates vulnerabilities by normalized severity. An advisory
// carrying a recognized-but-nonstandard label is counted as Info so it remains
// visible without being overstated. An advisory whose severity could not be
// determined at all — an absent label or a failed/truncated detail fetch — is
// counted as Unknown, which consumers must treat as fail-closed: it could be
// anything up to critical, so it must not read as harmless.
type SeverityCounts struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Moderate int `json:"moderate"`
	Low      int `json:"low"`
	Info     int `json:"info"`
	Unknown  int `json:"unknown"`
}

// Total returns the sum of all severity buckets.
func (c SeverityCounts) Total() int {
	return c.Critical + c.High + c.Moderate + c.Low + c.Info + c.Unknown
}

// Vulnerability is one advisory tied to the package that triggered it.
type Vulnerability struct {
	ID          string `json:"id"`
	Package     string `json:"package"`
	Version     string `json:"version"`
	Ecosystem   string `json:"ecosystem"`
	Severity    string `json:"severity"` // normalized: critical|high|moderate|low|info|unknown
	FixedIn     string `json:"fixedIn,omitempty"`
	AdvisoryURL string `json:"advisoryUrl,omitempty"`
	Summary     string `json:"summary,omitempty"`
}

// Result is the outcome of scanning a single lock file.
type Result struct {
	LockFile        string          `json:"lockFile"`
	Ecosystem       string          `json:"ecosystem"`
	Dependencies    int             `json:"dependencies"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	Counts          SeverityCounts  `json:"counts"`
}

// Scanner queries OSV.dev for vulnerabilities affecting pinned dependencies.
// BaseURL and HTTPClient are exported so callers (and tests) can point the
// scanner at an alternate endpoint or supply a custom client; the zero value is
// not usable — construct one with New.
type Scanner struct {
	BaseURL    string
	HTTPClient *http.Client
}

// New returns a Scanner configured against the public OSV.dev endpoint with a
// bounded HTTP timeout.
func New() *Scanner {
	return &Scanner{
		BaseURL:    DefaultOSVBaseURL,
		HTTPClient: &http.Client{Timeout: scanHTTPTimeout},
	}
}

// client returns the configured HTTP client, defaulting to a bounded one so a
// zero-value or partially-constructed Scanner never dereferences a nil client.
func (s *Scanner) client() *http.Client {
	if s.HTTPClient != nil {
		return s.HTTPClient
	}
	return &http.Client{Timeout: scanHTTPTimeout}
}

// baseURL returns the configured OSV base URL, defaulting to the public
// endpoint.
func (s *Scanner) baseURL() string {
	if s.BaseURL != "" {
		return s.BaseURL
	}
	return DefaultOSVBaseURL
}

// ScanFile scans a single lock file at lockPath, resolving the parser and OSV
// ecosystem from the file's base name. It returns (nil, nil) when the file is
// not a lock format with OSV coverage — the caller then knows the ecosystem was
// not scanned rather than confirmed vulnerability-free. A lock file that cannot
// be parsed yields an error wrapping ErrLockParse, and one with no pinned
// dependencies an error wrapping ErrNoPinnedDeps.
func (s *Scanner) ScanFile(ctx context.Context, lockPath string) (*Result, error) {
	lf, ok := LockFileForPath(lockPath)
	if !ok {
		return nil, nil
	}
	return s.scan(ctx, lf, lockPath)
}

// scan parses the lock file, queries OSV for the extracted dependencies, and
// aggregates the advisories into a Result.
func (s *Scanner) scan(ctx context.Context, lf LockFile, path string) (*Result, error) {
	pkgs, err := lf.parse(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %q: %w", ErrLockParse, path, err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("%s: %w", path, ErrNoPinnedDeps)
	}
	res := &Result{
		LockFile:     path,
		Ecosystem:    lf.ecosystem,
		Dependencies: len(pkgs),
	}

	ctx, cancel := context.WithTimeout(ctx, scanHTTPTimeout)
	defer cancel()

	idsByQuery, err := s.QueryBatch(ctx, pkgs)
	if err != nil {
		return nil, fmt.Errorf("querying OSV: %w", err)
	}

	res.Vulnerabilities = s.resolveVulns(ctx, pkgs, idsByQuery)
	for _, v := range res.Vulnerabilities {
		switch v.Severity {
		case "critical":
			res.Counts.Critical++
		case "high":
			res.Counts.High++
		case "moderate":
			res.Counts.Moderate++
		case "low":
			res.Counts.Low++
		case "unknown":
			res.Counts.Unknown++
		default:
			res.Counts.Info++
		}
	}
	return res, nil
}

// osvBatchRequest / osvBatchResponse mirror the OSV.dev /v1/querybatch contract.
type osvBatchRequest struct {
	Queries []osvQuery `json:"queries"`
}

type osvQuery struct {
	Package   osvQueryPackage `json:"package"`
	Version   string          `json:"version"`
	PageToken string          `json:"page_token,omitempty"`
}

type osvQueryPackage struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

type osvBatchResponse struct {
	Results []osvBatchResult `json:"results"`
}

// osvBatchResult is one query's slot of a /v1/querybatch response. A non-empty
// NextPageToken means OSV returned only the first page of that query's vulns.
type osvBatchResult struct {
	Vulns []struct {
		ID string `json:"id"`
	} `json:"vulns"`
	NextPageToken string `json:"next_page_token"`
}

// QueryBatch queries /v1/querybatch for the dependency set and returns, per
// query index, the vulnerability ids OSV reported. The slice is index-aligned
// with pkgs so callers can map a vuln back to the package that triggered it.
// The set is split into requests of at most maxBatchQueries (OSV's per-request
// limit), paginated results are followed to completion, and a response whose
// result count does not match its query count is an error — a short or
// mismatched response must never read as "no vulnerabilities". Callers own the
// deadline: bound ctx before calling.
func (s *Scanner) QueryBatch(ctx context.Context, pkgs []Package) ([][]string, error) {
	out := make([][]string, len(pkgs))
	for start := 0; start < len(pkgs); start += maxBatchQueries {
		end := min(start+maxBatchQueries, len(pkgs))
		if err := s.queryChunk(ctx, pkgs[start:end], out[start:end]); err != nil {
			return nil, fmt.Errorf("querying OSV for dependencies %d-%d of %d: %w", start+1, end, len(pkgs), err)
		}
	}
	return out, nil
}

// queryChunk resolves one chunk of at most maxBatchQueries packages into the
// index-aligned out slice, re-querying any result that carries a
// next_page_token (with that token) until every query's results are complete.
func (s *Scanner) queryChunk(ctx context.Context, pkgs []Package, out [][]string) error {
	queries := make([]osvQuery, len(pkgs))
	pending := make([]int, len(pkgs)) // indices whose results are still incomplete
	for i, p := range pkgs {
		queries[i] = osvQuery{
			Package: osvQueryPackage{Name: p.Name, Ecosystem: p.Ecosystem},
			Version: p.Version,
		}
		pending[i] = i
	}

	for round := 0; len(pending) > 0; round++ {
		if round == maxPageRounds {
			return fmt.Errorf("OSV results still paginated after %d requests", maxPageRounds)
		}
		batch := make([]osvQuery, len(pending))
		for j, idx := range pending {
			batch[j] = queries[idx]
		}
		results, err := s.postBatch(ctx, batch)
		if err != nil {
			return err
		}
		var next []int
		for j, idx := range pending {
			for _, v := range results[j].Vulns {
				out[idx] = append(out[idx], v.ID)
			}
			if tok := results[j].NextPageToken; tok != "" {
				queries[idx].PageToken = tok
				next = append(next, idx)
			}
		}
		pending = next
	}
	return nil
}

// postBatch sends one /v1/querybatch request and returns its results, which are
// guaranteed to be index-aligned with queries.
func (s *Scanner) postBatch(ctx context.Context, queries []osvQuery) ([]osvBatchResult, error) {
	body, err := json.Marshal(osvBatchRequest{Queries: queries})
	if err != nil {
		return nil, fmt.Errorf("marshaling OSV batch query: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL()+"/v1/querybatch", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building OSV request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling OSV: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OSV returned status %d", resp.StatusCode)
	}

	var parsed osvBatchResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBatchResponseBytes)).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding OSV response: %w", err)
	}
	if len(parsed.Results) != len(queries) {
		return nil, fmt.Errorf("OSV returned %d results for %d queries", len(parsed.Results), len(queries))
	}
	return parsed.Results, nil
}

// osvVuln models the subset of an OSV /v1/vulns/{id} record we surface. Severity
// is taken from database_specific.severity when present; when that is empty we
// fall back to the top-level CVSS "severity" array — Go advisories
// (GO-YYYY-NNNN) carry their severity only there, as CVSS vectors, and never in
// database_specific.severity. The top-level "details" text remains unmodeled
// (unused).
type osvVuln struct {
	ID         string        `json:"id"`
	Summary    string        `json:"summary"`
	Severity   []osvSeverity `json:"severity"`
	Affected   []osvAffected `json:"affected"`
	References []struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"references"`
	DatabaseSpecific struct {
		Severity string `json:"severity"`
	} `json:"database_specific"`
}

// osvAffected is one affected[] entry of an OSV record: the package it applies
// to and the version ranges (each a list of introduced/fixed events) in which
// that package is vulnerable. One record can cover several packages, and several
// release branches of one package.
type osvAffected struct {
	Package struct {
		Name      string `json:"name"`
		Ecosystem string `json:"ecosystem"`
	} `json:"package"`
	Ranges []struct {
		Type   string `json:"type"`
		Events []struct {
			Fixed string `json:"fixed"`
		} `json:"events"`
	} `json:"ranges"`
}

// osvSeverity is one entry of an OSV record's top-level severity[] array: a CVSS
// vector string (Score) tagged with its version (Type is CVSS_V2/CVSS_V3/CVSS_V4).
type osvSeverity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

// Detail is the per-advisory subset of an OSV /v1/vulns/{id} record consumers
// need to render or filter a finding. SeverityLabel is the raw
// database_specific.severity string (e.g. "CRITICAL", "MODERATE") when present,
// and otherwise the label derived from the record's top-level CVSS severity[]
// vectors (see severityFromCVSS); it is left unnormalized so consumers with
// different severity vocabularies can apply their own mapping. A zero-value
// Detail (failed or truncated fetch), or a record carrying neither source,
// leaves SeverityLabel empty, which NormalizeSeverity resolves to "unknown"
// (fail-closed), not "info". The fixed version depends on which package and
// version the advisory was matched against, so it is resolved per package with
// FixedInFor rather than stored once per advisory.
type Detail struct {
	ID            string
	Summary       string
	SeverityLabel string
	AdvisoryURL   string
	affected      []osvAffected
}

// newDetail builds the Detail for a fetched OSV record. It prefers
// database_specific.severity and falls back to the top-level CVSS severity[]
// vectors (the only source Go advisories carry) so their label is not left
// empty → "unknown". severityFromCVSS returns "" when nothing parses, preserving
// the fail-closed unknown path for records with neither source.
func newDetail(id string, v osvVuln) Detail {
	severity := v.DatabaseSpecific.Severity
	if severity == "" {
		severity = severityFromCVSS(v.Severity...)
	}
	return Detail{
		ID:            id,
		Summary:       v.Summary,
		SeverityLabel: severity,
		AdvisoryURL:   advisoryURL(v),
		affected:      v.Affected,
	}
}

// FixedInFor returns the version that fixes this advisory for pkg: the lowest
// "fixed" event above pkg.Version among the record's affected entries for pkg's
// own name and ecosystem. Entries for other packages, and fixes on older release
// branches that are not above the installed version, are ignored so the advice
// never names another package's version or a downgrade. It returns "" when the
// record names no applicable fix. Version ordering is best-effort across
// ecosystems (see compareVersions).
func (d Detail) FixedInFor(pkg Package) string {
	best := ""
	for _, a := range d.affected {
		if !affectsPackage(a, pkg) {
			continue
		}
		for _, r := range a.Ranges {
			if strings.EqualFold(r.Type, "GIT") {
				continue // commit hashes, not release versions
			}
			for _, e := range r.Events {
				if e.Fixed == "" {
					continue
				}
				if pkg.Version != "" && compareVersions(e.Fixed, pkg.Version) <= 0 {
					continue
				}
				if best == "" || compareVersions(e.Fixed, best) < 0 {
					best = e.Fixed
				}
			}
		}
	}
	return best
}

// affectsPackage reports whether an affected[] entry is about pkg. OSV
// ecosystems may carry a release suffix ("Debian:11"), so only the part before
// ":" is compared. PyPI names are compared in their PEP 503 normalized form.
// An entry that names no package is treated as applying.
func affectsPackage(a osvAffected, pkg Package) bool {
	if a.Package.Name == "" {
		return true
	}
	eco, _, _ := strings.Cut(a.Package.Ecosystem, ":")
	if !strings.EqualFold(eco, pkg.Ecosystem) {
		return false
	}
	if strings.EqualFold(pkg.Ecosystem, "PyPI") {
		return normalizePyPIName(a.Package.Name) == normalizePyPIName(pkg.Name)
	}
	return a.Package.Name == pkg.Name
}

// normalizePyPIName applies PEP 503 name normalization: lowercase, with runs of
// "-", "_" and "." collapsed to a single "-".
func normalizePyPIName(name string) string {
	var b strings.Builder
	sep := false
	for _, c := range strings.ToLower(name) {
		if c == '-' || c == '_' || c == '.' {
			sep = true
			continue
		}
		if sep && b.Len() > 0 {
			b.WriteByte('-')
		}
		sep = false
		b.WriteRune(c)
	}
	return b.String()
}

// resolveVulns fetches details for each unique vulnerability id, maps it back to
// the originating package, normalizes its severity, and returns the report list
// in deterministic order.
func (s *Scanner) resolveVulns(ctx context.Context, pkgs []Package, idsByQuery [][]string) []Vulnerability {
	details := s.FetchDetails(ctx, idsByQuery)

	var reports []Vulnerability
	for i, ids := range idsByQuery {
		for _, id := range ids {
			d := details[id]
			reports = append(reports, Vulnerability{
				ID:          id,
				Package:     pkgs[i].Name,
				Version:     pkgs[i].Version,
				Ecosystem:   pkgs[i].Ecosystem,
				Severity:    NormalizeSeverity(d.SeverityLabel),
				FixedIn:     d.FixedInFor(pkgs[i]),
				AdvisoryURL: d.AdvisoryURL,
				Summary:     d.Summary,
			})
		}
	}
	sort.Slice(reports, func(a, b int) bool {
		if reports[a].Package != reports[b].Package {
			return reports[a].Package < reports[b].Package
		}
		return reports[a].ID < reports[b].ID
	})
	return reports
}

// FetchDetails retrieves the OSV record for every unique vulnerability id (up
// to maxVulnDetailFetches), returning a map keyed by id. Ids are deduplicated
// and sorted before truncation so the fetched subset is deterministic. The
// fetches fan out across a bounded worker pool (each goroutine writes only its
// own index slot) so the whole set fits within the caller's deadline; the
// output stays deterministic because it is keyed by the pre-sorted ids. A
// failed fetch or an id dropped by truncation yields a zero-value Detail, whose
// empty SeverityLabel consumers map to their "unknown" bucket — the vuln is
// still reported, never silently dropped or given a fabricated severity.
// Callers own the deadline: bound ctx before calling.
func (s *Scanner) FetchDetails(ctx context.Context, idsByQuery [][]string) map[string]Detail {
	unique := make(map[string]struct{})
	for _, ids := range idsByQuery {
		for _, id := range ids {
			unique[id] = struct{}{}
		}
	}

	ids := make([]string, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	if len(ids) > maxVulnDetailFetches {
		slog.Warn("vulnscan: vulnerability detail fetch truncated; dropped vulns reported with unknown severity",
			"unique_vulns", len(ids), "fetch_limit", maxVulnDetailFetches)
		ids = ids[:maxVulnDetailFetches]
	}

	slots := make([]Detail, len(ids))
	sem := make(chan struct{}, detailFetchConcurrency)
	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			v, err := s.fetchVuln(ctx, id)
			if err != nil {
				slots[i] = Detail{ID: id}
				return
			}
			slots[i] = newDetail(id, v)
		}(i, id)
	}
	wg.Wait()

	details := make(map[string]Detail, len(ids))
	for i, id := range ids {
		details[id] = slots[i]
	}
	return details
}

// fetchVuln GETs a single /v1/vulns/{id} record. The id comes from an OSV
// response, so it is path-escaped: a "/", "?" or ".." in it cannot change the
// request target.
func (s *Scanner) fetchVuln(ctx context.Context, id string) (osvVuln, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL()+"/v1/vulns/"+url.PathEscape(id), nil)
	if err != nil {
		return osvVuln{}, err
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return osvVuln{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return osvVuln{}, fmt.Errorf("OSV vuln %s status %d", id, resp.StatusCode)
	}
	var v osvVuln
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxVulnResponseBytes)).Decode(&v); err != nil {
		return osvVuln{}, err
	}
	return v, nil
}

// NormalizeSeverity resolves a coarse severity bucket from the OSV record's
// database_specific.severity label. It is the single shared mapping used by both
// the CLI scan and the mcpserve security_scan tool, so the same advisory always
// reports the same severity. "medium" is folded into "moderate" (OSV GHSA
// advisories use MODERATE while some sources emit MEDIUM). An ABSENT label (empty
// string) — which is what a failed or truncated detail fetch leaves — yields
// "unknown", NOT "info": the severity is genuinely undetermined and must fail
// closed rather than be silently demoted below the exit gate. A present but
// unrecognized label yields "info" so the vuln stays visible without being
// overstated.
func NormalizeSeverity(label string) string {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "moderate", "medium":
		return "moderate"
	case "low":
		return "low"
	case "":
		return "unknown"
	default:
		return "info"
	}
}

// severityOrder ranks normalized severities for threshold comparison. "unknown"
// ranks at the top so an unresolved severity is always reported (fail-closed).
var severityOrder = map[string]int{
	"info": 0, "low": 1, "moderate": 2, "high": 3, "critical": 4, "unknown": 5,
}

// SeverityAtOrAbove reports whether a normalized severity meets or exceeds the
// given normalized threshold. Unknown always qualifies (fail-closed): an
// unresolved severity could be anything up to critical, so it is never filtered
// out by a threshold.
func SeverityAtOrAbove(severity, threshold string) bool {
	return severityOrder[severity] >= severityOrder[threshold]
}

// advisoryURL returns the canonical advisory link, preferring a reference typed
// ADVISORY and falling back to the first reference.
func advisoryURL(v osvVuln) string {
	for _, r := range v.References {
		if strings.EqualFold(r.Type, "ADVISORY") {
			return r.URL
		}
	}
	if len(v.References) > 0 {
		return v.References[0].URL
	}
	return ""
}
