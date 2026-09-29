package vulnscan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

// TestScanFile_NoPinnedDeps proves a lock file that yields no pinned
// coordinates is reported as ErrNoPinnedDeps rather than as a successful,
// vulnerability-free scan: nothing was sent to OSV.
func TestScanFile_NoPinnedDeps(t *testing.T) {
	t.Parallel()
	lock := writeLock(t, t.TempDir(), "requirements.txt", "requests>=2.0\nflask\n")

	s := New() // never contacts the network: there is nothing to query
	res, err := s.ScanFile(context.Background(), lock)
	if !errors.Is(err, ErrNoPinnedDeps) {
		t.Fatalf("ScanFile error = %v, want ErrNoPinnedDeps", err)
	}
	if res != nil {
		t.Errorf("result = %+v, want nil", res)
	}
}

// TestScanFile_ParseError proves a present but malformed lock file surfaces
// ErrLockParse so callers can tell it apart from an OSV failure.
func TestScanFile_ParseError(t *testing.T) {
	t.Parallel()
	lock := writeLock(t, t.TempDir(), "package-lock.json", "{not json")

	_, err := New().ScanFile(context.Background(), lock)
	if !errors.Is(err, ErrLockParse) {
		t.Fatalf("ScanFile error = %v, want ErrLockParse", err)
	}
}

// batchServer returns a Scanner backed by an OSV stub whose /v1/querybatch
// handler is supplied by the test, for response shapes the shared vulnscantest
// stub never produces (short results, pagination). Like OSV.dev, the stub
// rejects a request carrying more than maxBatchQueries queries.
func batchServer(t *testing.T, handle func(queries []osvQuery) osvBatchResponse) *Scanner {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req osvBatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(req.Queries) > maxBatchQueries {
			http.Error(w, `{"code":3,"message":"too many queries"}`, http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(handle(req.Queries))
	}))
	t.Cleanup(srv.Close)
	return &Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()}
}

// withVulns returns a batch result listing the given vulnerability ids.
func withVulns(ids ...string) osvBatchResult {
	var r osvBatchResult
	for _, id := range ids {
		r.Vulns = append(r.Vulns, struct {
			ID string `json:"id"`
		}{ID: id})
	}
	return r
}

func testPackages(n int) []Package {
	pkgs := make([]Package, n)
	for i := range pkgs {
		pkgs[i] = Package{Name: fmt.Sprintf("pkg-%d", i), Version: "1.0.0", Ecosystem: "npm"}
	}
	return pkgs
}

// TestQueryBatch_RejectsMismatchedResults proves a response carrying fewer (or
// more) results than queries is an error, not a clean result for the queries
// left without a slot.
func TestQueryBatch_RejectsMismatchedResults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		delta int
	}{
		{"empty results", -2},
		{"short results", -1},
		{"extra results", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := batchServer(t, func(q []osvQuery) osvBatchResponse {
				return osvBatchResponse{Results: make([]osvBatchResult, len(q)+tt.delta)}
			})
			got, err := s.QueryBatch(context.Background(), testPackages(2))
			if err == nil {
				t.Fatalf("QueryBatch = %v with nil error, want a result-count mismatch error", got)
			}
		})
	}
}

// TestQueryBatch_ChunksToOSVLimit proves a dependency set larger than OSV's
// per-request limit is split into requests of at most maxBatchQueries, and that
// the merged results stay index-aligned with the input packages.
func TestQueryBatch_ChunksToOSVLimit(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var sizes []int
	s := batchServer(t, func(q []osvQuery) osvBatchResponse {
		mu.Lock()
		sizes = append(sizes, len(q))
		mu.Unlock()
		resp := osvBatchResponse{Results: make([]osvBatchResult, len(q))}
		for i, query := range q {
			resp.Results[i] = withVulns("V-" + query.Package.Name)
		}
		return resp
	})

	pkgs := testPackages(2500)
	got, err := s.QueryBatch(context.Background(), pkgs)
	if err != nil {
		t.Fatalf("QueryBatch: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if want := []int{1000, 1000, 500}; fmt.Sprint(sizes) != fmt.Sprint(want) {
		t.Errorf("request sizes = %v, want %v", sizes, want)
	}
	for i, p := range pkgs {
		if len(got[i]) != 1 || got[i][0] != "V-"+p.Name {
			t.Fatalf("result %d = %v, want [V-%s] (index alignment lost)", i, got[i], p.Name)
		}
	}
}

// TestQueryBatch_FollowsPageToken proves a result carrying next_page_token is
// re-queried with that token and its pages are merged, rather than truncated.
func TestQueryBatch_FollowsPageToken(t *testing.T) {
	t.Parallel()
	s := batchServer(t, func(q []osvQuery) osvBatchResponse {
		resp := osvBatchResponse{Results: make([]osvBatchResult, len(q))}
		for i, query := range q {
			switch {
			case query.Package.Name != "pkg-1":
				resp.Results[i] = withVulns("A-" + query.Package.Name)
			case query.PageToken == "":
				resp.Results[i] = withVulns("P1")
				resp.Results[i].NextPageToken = "page-2"
			case query.PageToken == "page-2":
				resp.Results[i] = withVulns("P2")
			}
		}
		return resp
	})

	got, err := s.QueryBatch(context.Background(), testPackages(3))
	if err != nil {
		t.Fatalf("QueryBatch: %v", err)
	}
	want := [][]string{{"A-pkg-0"}, {"P1", "P2"}, {"A-pkg-2"}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("QueryBatch = %v, want %v", got, want)
	}
}

// TestFetchDetails_EscapesID proves a vulnerability id is path-escaped into the
// /v1/vulns/ request path, so an id containing "/", "?" or ".." cannot change
// the request target.
func TestFetchDetails_EscapesID(t *testing.T) {
	t.Parallel()
	const id = "../../internal/x?y"
	var mu sync.Mutex
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.EscapedPath()
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(osvVuln{ID: id})
	}))
	t.Cleanup(srv.Close)
	s := &Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()}

	s.FetchDetails(context.Background(), [][]string{{id}})
	mu.Lock()
	defer mu.Unlock()
	if want := "/v1/vulns/" + url.PathEscape(id); gotPath != want {
		t.Errorf("request path = %q, want %q", gotPath, want)
	}
}

// TestDetailFixedInFor proves the fixed version is chosen for the scanned
// package and its release branch: entries for other packages are ignored, and
// the lowest fix above the installed version wins.
func TestDetailFixedInFor(t *testing.T) {
	t.Parallel()
	const record = `{
  "id": "GHSA-test",
  "affected": [
    {"package": {"name": "foo-plugin", "ecosystem": "npm"},
     "ranges": [{"type": "SEMVER", "events": [{"introduced": "0"}, {"fixed": "0.1.0"}]}]},
    {"package": {"name": "foo", "ecosystem": "npm"},
     "ranges": [
       {"type": "SEMVER", "events": [{"introduced": "1.0.0"}, {"fixed": "1.2.9"}]},
       {"type": "SEMVER", "events": [{"introduced": "2.0.0"}, {"fixed": "2.0.3"}]},
       {"type": "GIT", "events": [{"introduced": "0"}, {"fixed": "abc123"}]}
     ]},
    {"package": {"name": "Django_Rest", "ecosystem": "PyPI"},
     "ranges": [{"type": "ECOSYSTEM", "events": [{"introduced": "0"}, {"fixed": "3.1.post1"}]}]}
  ]
}`
	var v osvVuln
	if err := json.Unmarshal([]byte(record), &v); err != nil {
		t.Fatalf("decoding record: %v", err)
	}
	d := newDetail(v.ID, v)

	tests := []struct {
		name string
		pkg  Package
		want string
	}{
		{"2.x branch", Package{Name: "foo", Version: "2.0.1", Ecosystem: "npm"}, "2.0.3"},
		{"1.x branch", Package{Name: "foo", Version: "1.1.0", Ecosystem: "npm"}, "1.2.9"},
		{"other package in same record", Package{Name: "foo-plugin", Version: "0.0.5", Ecosystem: "npm"}, "0.1.0"},
		{"package not in record", Package{Name: "bar", Version: "1.0.0", Ecosystem: "npm"}, ""},
		{"ecosystem mismatch", Package{Name: "foo", Version: "1.1.0", Ecosystem: "PyPI"}, ""},
		{"pypi normalized name", Package{Name: "django-rest", Version: "3.1", Ecosystem: "PyPI"}, "3.1.post1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := d.FixedInFor(tt.pkg); got != tt.want {
				t.Errorf("FixedInFor(%+v) = %q, want %q", tt.pkg, got, tt.want)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		a, b string
		want int
	}{
		{"1.2.9", "2.0.1", -1},
		{"2.0.3", "2.0.1", 1},
		{"1.10.0", "1.9.0", 1},
		{"v1.2.3", "1.2.3", 0},
		{"1.0.0-rc1", "1.0.0", -1},
		{"1.0.0", "1.0.0-rc1", 1},
		{"1.0", "1.0.1", -1},
		{"1.0.post1", "1.0", 1},
		{"1.0.post1", "1.0.1", -1},
		{"1.0.0+build5", "1.0.0", 0},
		{"0.0.0-20210809222454-d867a43fc93e", "0.0.0-20220101000000-aaaaaaaaaaaa", -1},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			t.Parallel()
			if got := compareVersions(tt.a, tt.b); got != tt.want {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
