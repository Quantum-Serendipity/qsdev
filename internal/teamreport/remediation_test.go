package teamreport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

// stubGH replaces runGH for one test. It is not safe for parallel tests.
func stubGH(t *testing.T, fn func(ctx context.Context, args ...string) ([]byte, error)) {
	t.Helper()
	orig := runGH
	runGH = fn
	t.Cleanup(func() { runGH = orig })
}

// argValue returns the value following flag in args.
func argValue(args []string, flag string) string {
	i := slices.Index(args, flag)
	if i < 0 || i+1 >= len(args) {
		return ""
	}
	return args[i+1]
}

func writeScope(t *testing.T, repos ...string) string {
	t.Helper()
	scope := ScopeFile{}
	for _, r := range repos {
		scope.Projects = append(scope.Projects, ScopeProject{Repo: r})
	}
	data, err := json.Marshal(scope)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "scope.json")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestPerProjectArtifactMatchesDownloadPattern is the F326 regression test:
// the artifact the per-project steps upload must be matched by what both the
// team workflow and --scope collection download.
func TestPerProjectArtifactMatchesDownloadPattern(t *testing.T) {
	t.Parallel()
	m := regexp.MustCompile(`(?m)^\s*name: (` + regexp.QuoteMeta(postureArtifactPrefix) + `.+)$`).
		FindStringSubmatch(GeneratePerProjectSteps())
	if m == nil {
		t.Fatal("per-project steps upload no posture-report artifact")
	}
	uploaded := strings.NewReplacer(
		"${{ github.repository_owner }}", "acme",
		"${{ github.event.repository.name }}", "web-app",
	).Replace(m[1])

	scopePattern := argValue(scopeDownloadArgs("acme/web-app", "/tmp/x"), "--pattern")
	if ok, err := path.Match(scopePattern, uploaded); err != nil || !ok {
		t.Errorf("--scope download pattern %q does not match uploaded artifact %q", scopePattern, uploaded)
	}
	if !strings.Contains(GenerateTeamWorkflow(), "--pattern '"+postureArtifactPattern+"'") {
		t.Errorf("team workflow does not download %q", postureArtifactPattern)
	}
}

// TestCollectFromScope_TagsRepository covers F325/F326: every downloaded report
// is tagged with the repository it came from, so issues can be filed there.
func TestCollectFromScope_TagsRepository(t *testing.T) {
	stubGH(t, func(_ context.Context, args ...string) ([]byte, error) {
		if argValue(args, "--pattern") != postureArtifactPattern {
			t.Errorf("gh args %v do not download by %q", args, postureArtifactPattern)
		}
		dir := filepath.Join(argValue(args, "--dir"), postureArtifactPrefix+"x")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		r := makeReport("proj", 80, true, true, 0, 0, "v1.0.0", time.Now().UTC())
		data, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}
		return nil, os.WriteFile(filepath.Join(dir, "posture-report.json"), data, 0o644)
	})

	reports, warnings, err := CollectFromScope(context.Background(), writeScope(t, "acme/web-app"))
	if err != nil {
		t.Fatalf("CollectFromScope: %v (warnings %v)", err, warnings)
	}
	if len(reports) != 1 || reports[0].Repository != "acme/web-app" {
		t.Fatalf("reports = %+v, want one tagged acme/web-app", reports)
	}
}

// TestCollectFromScope_StopsOnCancel is the F555 regression test: a cancelled
// context stops the per-repository loop instead of moving on to the next repo.
func TestCollectFromScope_StopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	stubGH(t, func(context.Context, ...string) ([]byte, error) {
		calls++
		cancel() // Ctrl-C while the first download runs.
		return nil, errors.New("signal: interrupt")
	})

	_, _, err := CollectFromScope(ctx, writeScope(t, "acme/a", "acme/b", "acme/c"))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Errorf("gh ran %d times after cancellation, want 1", calls)
	}
}

// TestCreateIssuesViaCLI_ReportsWhatWasCreated is the F325 regression test: an
// issue without a repository is reported, not skipped as a silent success,
// and the created count reflects reality.
func TestCreateIssuesViaCLI_ReportsWhatWasCreated(t *testing.T) {
	var repos []string
	stubGH(t, func(_ context.Context, args ...string) ([]byte, error) {
		repos = append(repos, argValue(args, "--repo"))
		return nil, nil
	})

	created, err := CreateIssuesViaCLI(context.Background(), []IssueSpec{
		{Project: "known", Repo: "acme/known", Title: "t"},
		{Project: "orphan", Title: "t"},
	})
	if created != 1 || !slices.Equal(repos, []string{"acme/known"}) {
		t.Errorf("created = %d via %v, want 1 via [acme/known]", created, repos)
	}
	if !errors.Is(err, ErrIssueRepoUnknown) || !strings.Contains(err.Error(), "orphan") {
		t.Errorf("err = %v, want ErrIssueRepoUnknown naming orphan", err)
	}
}

// TestGenerateIssues_CarriesRepository is the F325 end-to-end check: a report
// that knows its repository yields an issue targeting it.
func TestGenerateIssues_CarriesRepository(t *testing.T) {
	t.Parallel()
	r := makeReport("proj", 40, false, false, 2, 0, "v1.0.0", time.Now().UTC())
	r.Repository = "acme/proj"
	tr, err := Aggregate([]*posture.PostureReport{r}, AggregateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	issues := GenerateIssues(tr, nil)
	if len(issues) != 1 || issues[0].Repo != "acme/proj" || issues[0].Project != "proj" {
		t.Errorf("issues = %+v, want one for proj in acme/proj", issues)
	}
}

// TestAggregate_ScanDateNotReportDate is the F336 regression test: freshness
// comes from the dependency scan, and never-scanned projects are flagged
// rather than shown as 0/0 with no alert.
func TestAggregate_ScanDateNotReportDate(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()

	unscanned := makeReport("unscanned", 90, true, true, 0, 0, "v1.0.0", now)
	unscanned.Dependencies.Scanned = false
	unscanned.Dependencies.LastScan = nil

	oldScan := now.Add(-10 * 24 * time.Hour)
	staleScan := makeReport("stale-scan", 90, true, true, 0, 0, "v1.0.0", now)
	staleScan.Dependencies.LastScan = &oldScan

	tr, err := Aggregate([]*posture.PostureReport{unscanned, staleScan}, AggregateOptions{QsdevVersion: "v1.0.0"})
	if err != nil {
		t.Fatal(err)
	}

	byName := map[string]ProjectSummary{}
	for _, p := range tr.Projects {
		byName[p.Name] = p
	}
	if p := byName["unscanned"]; p.Scanned || p.LastScan != nil || p.Stale {
		t.Errorf("unscanned summary = %+v, want Scanned=false, LastScan=nil, Stale=false", p)
	}
	if p := byName["stale-scan"]; !p.Stale {
		t.Errorf("a 10-day-old scan in a fresh report must be stale: %+v", p)
	}
	if tr.Summary.UnscannedProjects != 1 {
		t.Errorf("UnscannedProjects = %d, want 1", tr.Summary.UnscannedProjects)
	}

	var alerted bool
	for _, a := range tr.Alerts {
		if a.Project == "unscanned" && a.Severity == SeverityHigh && strings.Contains(a.Message, "not scanned") {
			alerted = true
		}
	}
	if !alerted {
		t.Errorf("no High 'not scanned' alert for the unscanned project: %+v", tr.Alerts)
	}

	md := RenderMarkdown(tr)
	if !strings.Contains(md, "| unscanned | 90.0 | A- | PASS | PASS | n/a | v1.0.0 | never |") {
		t.Errorf("unscanned row should show n/a vulns and never scanned:\n%s", md)
	}
}

// TestLoadPostureReports_SchemaMajorCompatible is the F339 regression test:
// reports from other minor schema versions load; other majors do not.
func TestLoadPostureReports_SchemaMajorCompatible(t *testing.T) {
	t.Parallel()
	major := schemaMajor(posture.SchemaVersion)
	tests := []struct {
		version string
		want    bool
	}{
		{version: major + ".0.0", want: true},
		{version: major + ".99.0", want: true},
		{version: "999.0.0", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			r := makeReport("p", 80, true, true, 0, 0, "v1.0.0", time.Now().UTC())
			r.SchemaVersion = tt.version
			data, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "r.json"), data, 0o644); err != nil {
				t.Fatal(err)
			}
			reports, warnings, err := LoadPostureReports(dir)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(reports) == 1; got != tt.want {
				t.Errorf("schema %s loaded = %v, want %v (warnings %v)", tt.version, got, tt.want, warnings)
			}
		})
	}
}

// TestAlertBelowThreshold is the F347 regression test for --threshold.
func TestAlertBelowThreshold(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	projects := []ProjectSummary{
		projectSummaryHelper("low", 60, true, true, 0, 0, "v1.0.0", now),
		projectSummaryHelper("high", 95, true, true, 0, 0, "v1.0.0", now),
	}
	tests := []struct {
		threshold float64
		want      []string
	}{
		{threshold: 70, want: []string{"low"}},
		{threshold: 0, want: nil},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("threshold %.0f", tt.threshold), func(t *testing.T) {
			t.Parallel()
			alerts := generateAlerts(projects, AggregateOptions{QsdevVersion: "v1.0.0", Threshold: tt.threshold}, nil)
			var got []string
			for _, a := range alerts {
				if strings.Contains(a.Message, "below the team threshold") {
					got = append(got, a.Project)
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("below-threshold alerts for %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAggregate_PrunesHistory is the F347 regression test: aggregation with
// trends drops history points older than the retention window.
func TestAggregate_PrunesHistory(t *testing.T) {
	t.Parallel()
	histPath := filepath.Join(t.TempDir(), "history.json")
	old := time.Now().UTC().Add(-historyRetention - 48*time.Hour).Format("2006-01-02")
	seed := &HistoryStore{Entries: map[string][]TrendPoint{
		"proj":    {{Date: old, Score: 50}},
		"retired": {{Date: old, Score: 40}},
	}}
	if err := SaveHistory(histPath, seed); err != nil {
		t.Fatal(err)
	}

	r := makeReport("proj", 80, true, true, 0, 0, "v1.0.0", time.Now().UTC())
	if _, err := Aggregate([]*posture.PostureReport{r}, AggregateOptions{HistoryFile: histPath, IncludeTrends: true}); err != nil {
		t.Fatal(err)
	}

	saved, err := LoadHistory(histPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := saved.Entries["retired"]; ok {
		t.Error("expired project history was not pruned")
	}
	for _, pt := range saved.Entries["proj"] {
		if pt.Date == old {
			t.Errorf("expired point %s kept for proj", old)
		}
	}
}
