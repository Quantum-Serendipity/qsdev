package container

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// OutputFormat selects the migration report output format.
type OutputFormat string

const (
	FormatText OutputFormat = "text"
	FormatJSON OutputFormat = "json"
)

// FormatMigrationReport writes a MigrationReport to w in the given format.
func FormatMigrationReport(report *MigrationReport, format OutputFormat, w io.Writer, useColor bool) error {
	switch format {
	case FormatJSON:
		return formatJSON(report, w)
	default:
		return formatText(report, w, useColor)
	}
}

func formatJSON(report *MigrationReport, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func formatText(report *MigrationReport, w io.Writer, useColor bool) error {
	// Header.
	fmt.Fprintf(w, "Container Migration Report\n")
	fmt.Fprintf(w, "==========================\n\n")
	fmt.Fprintf(w, "Source: %s  Target: %s\n", report.SourceRuntime, report.TargetRuntime)

	if report.RuntimeInfo != nil {
		fmt.Fprintf(w, "Active runtime: %s", report.RuntimeInfo.Active)
		if report.RuntimeInfo.Version != "" {
			fmt.Fprintf(w, " (%s)", report.RuntimeInfo.Version)
		}
		fmt.Fprintln(w)
	}

	if len(report.ComposeFiles) == 0 {
		fmt.Fprintln(w, "\nNo compose files found.")
		return nil
	}

	fmt.Fprintf(w, "Compose files: %d\n\n", len(report.ComposeFiles))

	if len(report.Issues) == 0 {
		fmt.Fprintln(w, "No migration issues found. The project is ready for Podman.")
		return nil
	}

	// Group issues by file, then by service.
	type issueGroup struct {
		file     string
		services map[string][]MigrationIssue
		order    []string // preserve service encounter order
	}
	groups := make(map[string]*issueGroup)
	var fileOrder []string

	for _, issue := range report.Issues {
		g, ok := groups[issue.File]
		if !ok {
			g = &issueGroup{
				file:     issue.File,
				services: make(map[string][]MigrationIssue),
			}
			groups[issue.File] = g
			fileOrder = append(fileOrder, issue.File)
		}
		svc := issue.Service
		if svc == "" {
			svc = "(global)"
		}
		if _, exists := g.services[svc]; !exists {
			g.order = append(g.order, svc)
		}
		g.services[svc] = append(g.services[svc], issue)
	}

	for _, file := range fileOrder {
		g := groups[file]
		relPath := filepath.Base(g.file)
		fmt.Fprintf(w, "--- %s ---\n", relPath)
		for _, svc := range g.order {
			issues := g.services[svc]
			fmt.Fprintf(w, "  [%s]\n", svc)
			for _, issue := range issues {
				symbol := severitySymbol(issue.Severity, useColor)
				fixable := ""
				if issue.AutoFixable {
					fixable = " [auto-fixable]"
				}
				fmt.Fprintf(w, "    %s %s%s\n", symbol, issue.Description, fixable)
			}
		}
		fmt.Fprintln(w)
	}

	// Summary.
	s := report.Summary
	fmt.Fprintf(w, "Summary: %d issue(s)", s.Total)
	parts := []string{}
	if s.Critical > 0 {
		parts = append(parts, fmt.Sprintf("%d critical", s.Critical))
	}
	if s.Warning > 0 {
		parts = append(parts, fmt.Sprintf("%d warning", s.Warning))
	}
	if s.Info > 0 {
		parts = append(parts, fmt.Sprintf("%d info", s.Info))
	}
	if len(parts) > 0 {
		fmt.Fprintf(w, " (%s)", strings.Join(parts, ", "))
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Auto-fixable: %d  Manual: %d\n", s.AutoFixable, s.ManualOnly)

	return nil
}

// severitySymbol returns a colored symbol for terminal output.
func severitySymbol(sev IssueSeverity, useColor bool) string {
	if useColor {
		switch sev {
		case SeverityCritical:
			return "\033[31m[CRITICAL]\033[0m"
		case SeverityWarning:
			return "\033[33m[WARNING]\033[0m"
		case SeverityInfo:
			return "\033[36m[INFO]\033[0m"
		}
	}
	switch sev {
	case SeverityCritical:
		return "[CRITICAL]"
	case SeverityWarning:
		return "[WARNING]"
	case SeverityInfo:
		return "[INFO]"
	}
	return "[UNKNOWN]"
}
