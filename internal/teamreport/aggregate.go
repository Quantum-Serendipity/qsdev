package teamreport

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/posture"
)

const staleScanThreshold = 7 * 24 * time.Hour

// LoadPostureReports walks dir recursively for .json files, deserializes
// each as a PostureReport, and validates the schemaVersion field. It returns
// the successfully loaded reports and a list of warning messages for files
// that could not be loaded.
func LoadPostureReports(dir string) ([]*posture.PostureReport, []string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("accessing report directory: %w", err)
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("not a directory: %s", dir)
	}

	var reports []*posture.PostureReport
	var warnings []string

	walkErr := filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipping %s: %v", path, err))
			return nil
		}
		if fi.IsDir() {
			return nil
		}
		if !strings.HasSuffix(fi.Name(), ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to read %s: %v", path, err))
			return nil
		}

		var report posture.PostureReport
		if err := json.Unmarshal(data, &report); err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to parse %s: %v", path, err))
			return nil
		}

		if report.SchemaVersion == "" {
			warnings = append(warnings, fmt.Sprintf("skipping %s: missing schemaVersion", path))
			return nil
		}
		if report.SchemaVersion != posture.SchemaVersion {
			warnings = append(warnings, fmt.Sprintf(
				"skipping %s: unsupported schema version %q (expected %q)",
				path, report.SchemaVersion, posture.SchemaVersion,
			))
			return nil
		}

		reports = append(reports, &report)
		return nil
	})
	if walkErr != nil {
		return reports, warnings, fmt.Errorf("walking report directory: %w", walkErr)
	}

	return reports, warnings, nil
}

// Aggregate converts a set of PostureReports into a TeamReport with computed
// summary statistics, alerts, and optional trend data.
func Aggregate(reports []*posture.PostureReport, opts AggregateOptions) (*TeamReport, error) {
	if len(reports) == 0 {
		return nil, errors.New("no posture reports to aggregate")
	}

	now := time.Now().UTC()
	projects := make([]ProjectSummary, 0, len(reports))

	for _, r := range reports {
		ps := ProjectSummary{
			Name:        r.ProjectName,
			Score:       r.Score,
			Conformance: r.Conformance,
			VulnTotals:  r.Dependencies.Totals,
			GdevVersion: r.GdevVersion,
			LastScan:    r.GeneratedAt,
		}

		// Mark as stale if the scan is older than the threshold.
		if now.Sub(r.GeneratedAt) > staleScanThreshold {
			ps.Stale = true
		}

		projects = append(projects, ps)
	}

	summary := computeSummary(projects, opts)

	// Load history if available.
	var history *HistoryStore
	if opts.HistoryFile != "" {
		var err error
		history, err = LoadHistory(opts.HistoryFile)
		if err != nil {
			return nil, fmt.Errorf("loading history: %w", err)
		}
	}

	// Generate alerts.
	alerts := generateAlerts(projects, opts, history)

	teamReport := &TeamReport{
		SchemaVersion: posture.SchemaVersion,
		GeneratedAt:   now,
		Summary:       summary,
		Projects:      projects,
		Alerts:        alerts,
	}

	// Add trends if history is available and requested.
	if opts.IncludeTrends && history != nil {
		// Append current scores to history.
		history.Append(projects)

		trends := make([]ProjectTrend, 0, len(history.Entries))
		for project, points := range history.Entries {
			trends = append(trends, ProjectTrend{
				Project:    project,
				DataPoints: points,
			})
		}
		// Sort trends by project name for deterministic output.
		sort.Slice(trends, func(i, j int) bool {
			return trends[i].Project < trends[j].Project
		})
		teamReport.Trends = trends

		// Save updated history.
		if err := SaveHistory(opts.HistoryFile, history); err != nil {
			return nil, fmt.Errorf("saving history: %w", err)
		}
	}

	return teamReport, nil
}

// computeSummary calculates aggregate statistics from the project summaries.
func computeSummary(projects []ProjectSummary, opts AggregateOptions) TeamSummary {
	n := len(projects)
	if n == 0 {
		return TeamSummary{}
	}

	var (
		totalScore     float64
		scores         []float64
		baselinePass   int
		enhancedPass   int
		criticalVulns  int
		highVulns      int
		needUpdate     int
	)

	for _, p := range projects {
		totalScore += p.Score.Total
		scores = append(scores, p.Score.Total)

		if p.Conformance.Baseline.Pass {
			baselinePass++
		}
		if p.Conformance.Enhanced.Pass {
			enhancedPass++
		}

		criticalVulns += p.VulnTotals.Critical
		highVulns += p.VulnTotals.High

		if isOutdatedGdev(p.GdevVersion, opts.GdevVersion) {
			needUpdate++
		}
	}

	sort.Float64s(scores)

	return TeamSummary{
		ProjectCount:       n,
		AverageScore:       roundTo1(totalScore / float64(n)),
		MedianScore:        roundTo1(medianFloat64(scores)),
		BaselinePassRate:   roundTo1(float64(baselinePass) / float64(n) * 100),
		EnhancedPassRate:   roundTo1(float64(enhancedPass) / float64(n) * 100),
		TotalCriticalVulns: criticalVulns,
		TotalHighVulns:     highVulns,
		ProjectsNeedUpdate: needUpdate,
	}
}

// isOutdatedGdev returns true if the project's gdev version is more than
// 2 minor versions behind the current version.
func isOutdatedGdev(projectVersion, currentVersion string) bool {
	if currentVersion == "" || projectVersion == "" {
		return false
	}

	projMajor, projMinor := parseVersion(projectVersion)
	currMajor, currMinor := parseVersion(currentVersion)

	if projMajor < currMajor {
		return true
	}
	if projMajor == currMajor && currMinor-projMinor > 2 {
		return true
	}
	return false
}

// parseVersion extracts major and minor version numbers from a semver-like
// string. Returns (0, 0) for unparseable versions.
func parseVersion(v string) (major, minor int) {
	// Strip leading "v" if present.
	v = strings.TrimPrefix(v, "v")

	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return 0, 0
	}

	_, _ = fmt.Sscanf(parts[0], "%d", &major)
	_, _ = fmt.Sscanf(parts[1], "%d", &minor)
	return major, minor
}
