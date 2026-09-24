package posture

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/internal/vulnscan"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// newVulnScanner constructs the dependency vulnerability scanner used when a
// fresh scan (AssessOptions.FreshScan) is requested. It is a package-level
// variable so tests can substitute a scanner pointed at a mock OSV endpoint
// without changing Assess's exported signature.
var newVulnScanner = func() *vulnscan.Scanner { return vulnscan.New() }

// StateFilesCategory is the drift category under which Assess records findings
// about qsdev's own state and config (state files that failed to load, unknown
// tool enablement, a malformed project config).
const StateFilesCategory = "state-files"

// ErrNotInitialized is returned when Assess is called on a project that has
// not been initialized with qsdev init (no state files or .qsdev.yaml found).
var ErrNotInitialized = errors.New("project not initialized: run 'qsdev init' first")

// Assess performs a security posture assessment of the project at projectPath.
// It loads all state files, checks that the project is initialized, and
// assembles a PostureReport. Returns ErrNotInitialized if no state files or
// .qsdev.yaml exist.
func Assess(projectPath string, opts AssessOptions) (*PostureReport, error) {
	// Validate that the project path exists.
	info, err := os.Stat(projectPath)
	if err != nil {
		return nil, fmt.Errorf("accessing project path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project path is not a directory: %s", projectPath)
	}

	// Load all state files.
	merged := LoadAllStates(projectPath)

	// Check for .qsdev.yaml as an alternative initialization indicator.
	qsdevYAMLExists := false
	if _, err := os.Stat(filepath.Join(projectPath, branding.Get().ConfigFile)); err == nil {
		qsdevYAMLExists = true
	}

	// If no state files were loaded and no .qsdev.yaml exists, the project
	// is not initialized.
	if !merged.HasAnyState() && !qsdevYAMLExists {
		return nil, ErrNotInitialized
	}

	// Determine project name from directory basename.
	projectName := filepath.Base(projectPath)

	// Get version info.
	buildInfo := version.Info()
	qsdevVersion := buildInfo.Version
	if merged.QsdevVersion != "" {
		qsdevVersion = merged.QsdevVersion
	}

	report := &PostureReport{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		QsdevVersion:  qsdevVersion,
		ProjectPath:   projectPath,
		ProjectName:   projectName,
		Repository:    ciRepository(),
		Score: AggregateScore{
			Grade: "U", // Unscored — will be filled by scoring logic.
		},
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{
				Checks: []ConformanceCheck{},
			},
			Enhanced: ConformanceLevel{
				Checks: []ConformanceCheck{},
			},
		},
		Defense: DefenseCoverage{
			Layers: []DefenseLayer{},
		},
		Config: ConfigHealth{
			Files: []ConfigFileInfo{},
		},
		Dependencies: DependencyHealth{
			Ecosystems: []EcosystemStatus{},
		},
		Drift: drift.Report{
			Categories: []drift.Category{},
			BySeverity: make(map[drift.Severity]int),
		},
		Tools:      []ToolStatus{},
		Ecosystems: []EcosystemStatus{},
	}

	// Determine progressive tier from config. A config that exists but cannot
	// be parsed must not silently grade the project against the default tier.
	configFile := branding.Get().ConfigFile
	currentTierName, cfgErr := resolveTierName(filepath.Join(projectPath, configFile))
	if cfgErr != nil {
		slog.Warn("posture: config unreadable; assessing against the default tier",
			"config", configFile, "tier", currentTierName, "error", cfgErr)
		addDriftFinding(report, StateFilesCategory, drift.Finding{
			Category: StateFilesCategory,
			Severity: drift.Error,
			Subject:  configFile,
			Description: fmt.Sprintf("Failed to parse %s; the tier defaults to %q, which may not match the project: %s",
				configFile, currentTierName, cfgErr),
			Remediation: fmt.Sprintf("Fix the error in %s.", configFile),
		})
	}
	nextTierName, _ := tier.NextTier(currentTierName)
	report.Tier = ReportTierInfo{
		Current:  currentTierName,
		Position: tier.Position(currentTierName),
		Total:    tier.Total(),
		NextTier: nextTierName,
	}

	// Record any state loading errors as drift findings.
	for _, loadErr := range merged.Errors {
		addDriftFinding(report, StateFilesCategory, drift.Finding{
			Category:    StateFilesCategory,
			Severity:    drift.Warning,
			Subject:     loadErr.Path,
			Description: fmt.Sprintf("Failed to load state file: %s", loadErr.Err),
			Remediation: "Re-run 'qsdev init' to regenerate state files.",
			AutoFixable: true,
		})
	}
	if merged.ToolStateUnknown {
		addDriftFinding(report, StateFilesCategory, drift.Finding{
			Category: StateFilesCategory,
			Severity: drift.Warning,
			Subject:  "enabled tools",
			Description: "Tool enablement is unknown: no state file or answers file records which tools " +
				"are enabled, so no tool is credited",
			Remediation: "Re-run 'qsdev init' to record the enabled tools.",
			AutoFixable: true,
		})
	}

	// Build inputs for the scoring pipeline. genState is the tracked state as
	// recorded; config health and drift detection compare exactly that against
	// the filesystem.
	enabledTools := merged.EnabledTools
	genState := types.GeneratedState{
		QsdevVersion: merged.QsdevVersion,
		Files:        merged.Files,
		EnabledTools: merged.EnabledTools,
	}
	detected := detect.Detect(context.Background(), projectPath)

	// Defense and conformance also credit protection the project has outside
	// qsdev's state, such as a hand-written pre-commit config. That goes into a
	// separate view so it never reaches config health or drift detection, which
	// would otherwise report an untracked file as a modified machine-owned one.
	activeTools, observedState := observedProtections(projectPath, enabledTools, genState)

	// Assess config health from tracked files.
	configFiles := buildConfigFileInfos(projectPath, merged.Files)
	if configFiles == nil {
		configFiles = []ConfigFileInfo{}
	}

	// Defense layers and conformance judge the files that are actually on
	// disk. A path the state file still lists after its file was deleted (or
	// became unreadable) is not a present control.
	presentState := genState
	presentState.Files = presentFiles(genState.Files, configFiles)
	// Protection observed outside the tracked state (see observedProtections)
	// is present by construction: it was just read from disk.
	for path, fileState := range observedState.Files {
		if _, tracked := genState.Files[path]; !tracked {
			presentState.Files[path] = fileState
		}
	}

	// Assess defense layers.
	report.Defense = AssessDefenseLayers(projectPath, activeTools, detected, presentState, report.Tier.Position)

	configScore := ComputeConfigScore(configFiles)
	report.Config = ConfigHealth{
		Score: configScore,
		Files: configFiles,
	}
	for _, f := range configFiles {
		report.Config.Total++
		switch f.State {
		case "current":
			report.Config.Current++
		case "modified":
			report.Config.Modified++
		case "outdated":
			report.Config.Outdated++
		case "missing":
			report.Config.Missing++
		case "corrupt":
			report.Config.Corrupt++
		}
	}

	// Assess dependency health. When a fresh scan is requested, run the OSV
	// scanner against each detected ecosystem's lock file so vulnerability counts
	// reflect reality rather than staying inertly zero.
	var scanner *vulnscan.Scanner
	if opts.FreshScan {
		scanner = newVulnScanner()
	}
	ecoStatuses := buildEcosystemStatuses(detected, projectPath, scanner)
	if ecoStatuses == nil {
		ecoStatuses = []EcosystemStatus{}
	}
	depHealth := ComputeDepScore(ecoStatuses)
	report.Dependencies = depHealth
	if report.Dependencies.Ecosystems == nil {
		report.Dependencies.Ecosystems = []EcosystemStatus{}
	}
	// Record whether a scan actually ran to completion. A requested scan whose
	// per-ecosystem checks errored (e.g. OSV unreachable) leaves zero Totals that
	// mean "unknown", not "clean"; likewise an ecosystem whose lock format has no
	// OSV coverage is never scanned at all. Derive the aggregate flags from the
	// actual per-ecosystem outcomes rather than the request flag, so conformance,
	// rendering, and the exit gate never present an unscanned or failed check as a
	// clean bill of health. Scanned requires that at least one ecosystem was in
	// fact scanned OK — a project with no OSV-covered ecosystem is "not scanned",
	// not "scanned clean".
	scanFailed := slices.ContainsFunc(ecoStatuses, func(e EcosystemStatus) bool {
		return e.ScanError
	})
	scannedAny := slices.ContainsFunc(ecoStatuses, func(e EcosystemStatus) bool {
		return e.Scanned
	})
	report.Dependencies.Scanned = opts.FreshScan && scannedAny && !scanFailed
	report.Dependencies.ScanFailed = opts.FreshScan && scanFailed
	if opts.FreshScan {
		now := time.Now().UTC()
		report.Dependencies.LastScan = &now
	}
	report.Ecosystems = ecoStatuses

	// Evaluate conformance.
	report.Conformance = EvaluateConformance(report.Defense, report.Dependencies, activeTools, presentState)

	// Run drift detection.
	driftReport := drift.Detect(projectPath, genState, enabledTools)
	for _, cat := range driftReport.Categories {
		for _, f := range cat.Findings {
			addDriftFinding(report, cat.Name, f)
		}
	}

	report.Tools = buildToolStatuses(enabledTools)

	// Compute aggregate score.
	report.Score = ComputeAggregateScore(report.Defense.Score, configScore, report.Dependencies.Score)

	return report, nil
}

// resolveTierName returns the project's progressive tier from its config file:
// the explicit tier, else one inferred from the Claude Code settings. An absent
// config yields the default tier. A config that exists but cannot be read or
// parsed yields the default together with the error, so the caller can report
// that the tier is a guess.
func resolveTierName(configPath string) (string, error) {
	cfg, err := config.ParseQsdevConfig(configPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return tier.Default().String(), nil
		}
		return tier.Default().String(), err
	}
	if cfg.Tier != "" {
		return cfg.Tier, nil
	}
	return tier.Infer(cfg.ClaudeCode.PermissionLevel, cfg.ClaudeCode.MCPServers).String(), nil
}

// addDriftFinding records a finding under the named category and updates the
// report's drift totals.
func addDriftFinding(report *PostureReport, category string, finding drift.Finding) {
	report.Drift.Categories = appendOrCreateCategory(report.Drift.Categories, category, finding)
	report.Drift.TotalFindings++
	report.Drift.BySeverity[finding.Severity]++
}

// repositorySlugRe matches a GitHub "owner/name" repository slug.
var repositorySlugRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9._-]+$`)

// ciRepository returns the "owner/name" repository GitHub Actions reports for
// the running workflow, so reports uploaded from CI identify their source
// repository for team aggregation. It returns "" outside GitHub Actions or
// when the value is not a well-formed slug.
func ciRepository() string {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		return ""
	}
	repo := os.Getenv("GITHUB_REPOSITORY")
	if !repositorySlugRe.MatchString(repo) {
		return ""
	}
	return repo
}

// appendOrCreateCategory adds a finding to the named category, creating the
// category if it does not already exist.
func appendOrCreateCategory(categories []drift.Category, name string, finding drift.Finding) []drift.Category {
	for i, cat := range categories {
		if cat.Name == name {
			categories[i].Findings = append(categories[i].Findings, finding)
			return categories
		}
	}
	return append(categories, drift.Category{
		Name:     name,
		Findings: []drift.Finding{finding},
	})
}

// buildConfigFileInfos compares tracked state files against the filesystem
// to determine each file's current health status.
func buildConfigFileInfos(projectPath string, files map[string]types.FileState) []ConfigFileInfo {
	var infos []ConfigFileInfo
	for path, stored := range files {
		info := ConfigFileInfo{
			Path:       path,
			Category:   FileCategory(stored.Strategy),
			StoredHash: stored.Hash,
		}
		absPath := filepath.Join(projectPath, path)
		currentHash, err := state.ComputeFileHash(absPath)
		if err != nil {
			// ComputeFileHash wraps the read error, which os.IsNotExist does
			// not unwrap; errors.Is does.
			if errors.Is(err, fs.ErrNotExist) {
				info.State = "missing"
			} else {
				info.State = "corrupt"
			}
		} else {
			info.CurrentHash = currentHash
			info.HashMatch = currentHash == stored.Hash
			if info.HashMatch {
				info.State = "current"
			} else {
				info.State = "modified"
			}
		}
		infos = append(infos, info)
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Path < infos[j].Path })
	return infos
}

// presentFiles returns the subset of tracked files whose config health shows
// them present and readable on disk (current or modified). Missing and corrupt
// files are left out.
func presentFiles(files map[string]types.FileState, infos []ConfigFileInfo) map[string]types.FileState {
	present := make(map[string]types.FileState, len(infos))
	for _, info := range infos {
		if info.State != "current" && info.State != "modified" {
			continue
		}
		if fs, ok := files[info.Path]; ok {
			present[info.Path] = fs
		}
	}
	return present
}

// buildEcosystemStatuses detects which ecosystems are present and whether
// their lock files exist. When scanner is non-nil (a fresh scan was requested),
// each present, OSV-covered lock file is scanned and its vulnerability counts
// populated on the returned EcosystemStatus. The per-ecosystem scans run
// concurrently — each writes only to its own slot — so the total scan time is
// the slowest ecosystem rather than the sum of all of them.
func buildEcosystemStatuses(detected types.DetectedProject, projectPath string, scanner *vulnscan.Scanner) []EcosystemStatus {
	var statuses []EcosystemStatus
	var lockPaths []string // index-aligned with statuses; "" when no lock file
	for name, present := range detected.Ecosystems {
		if !present {
			continue
		}
		status := EcosystemStatus{
			Name:     name,
			Detected: true,
		}
		// Prefer the lock file the scanner would choose (dedicated locks such as
		// poetry.lock before a loose requirements.txt), so a scan reads the
		// authoritative pins; fall back to any catalog lock file for display,
		// in the same dedicated-first order.
		lf, lockAbs, scannable := vulnscan.LockFileForEcosystem(projectPath, name)
		if scannable {
			status.LockFile = lf.Name()
		} else {
			for _, lf := range ecosystem.OrderedLockFiles(name) {
				absPath := filepath.Join(projectPath, lf)
				if _, err := os.Stat(absPath); err == nil {
					status.LockFile = lf
					lockAbs = absPath
					break
				}
			}
		}
		if status.LockFile == "" {
			if _, hasEntries := ecosystem.LockFilesByEcosystem[name]; hasEntries {
				status.LockFile = "missing"
			} else {
				status.LockFile = "n/a"
			}
		}
		statuses = append(statuses, status)
		lockPaths = append(lockPaths, lockAbs)
	}
	if scanner != nil {
		var wg sync.WaitGroup
		for i := range statuses {
			if lockPaths[i] == "" {
				continue
			}
			wg.Add(1)
			go func(status *EcosystemStatus, lockAbs string) {
				defer wg.Done()
				scanEcosystem(status, scanner, lockAbs)
			}(&statuses[i], lockPaths[i])
		}
		wg.Wait()
	}
	sort.Slice(statuses, func(i, j int) bool { return statuses[i].Name < statuses[j].Name })
	return statuses
}

// scanEcosystem runs the OSV scanner against a single lock file and populates
// the ecosystem's vulnerability counts. A scan failure, an unsupported lock
// format, or a lock file with no pinned dependencies leaves the ecosystem
// unscanned (Scanned stays false, counts stay zero) rather than falsely
// reporting it clean.
func scanEcosystem(status *EcosystemStatus, scanner *vulnscan.Scanner, lockAbs string) {
	res, err := scanner.ScanFile(context.Background(), lockAbs)
	if errors.Is(err, vulnscan.ErrNoPinnedDeps) {
		// Nothing pinned to query (e.g. a requirements.txt of loose specifiers):
		// the ecosystem was not scanned, which is not the same as scanned clean.
		slog.Warn("posture: lock file has no pinned dependencies; ecosystem left unscanned",
			"ecosystem", status.Name, "lockFile", status.LockFile)
		return
	}
	if err != nil {
		slog.Warn("posture: dependency vulnerability scan failed; ecosystem left unscanned",
			"ecosystem", status.Name, "lockFile", status.LockFile, "error", err)
		status.ScanError = true
		return
	}
	if res == nil {
		// The lock format has no OSV coverage; nothing was scanned.
		return
	}
	status.VulnCounts = VulnSeverityCounts{
		Critical: res.Counts.Critical,
		High:     res.Counts.High,
		Moderate: res.Counts.Moderate,
		Low:      res.Counts.Low,
		Info:     res.Counts.Info,
		Unknown:  res.Counts.Unknown,
	}
	now := time.Now().UTC()
	status.LastScan = &now
	status.Scanned = true
}
