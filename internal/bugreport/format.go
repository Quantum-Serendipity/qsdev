package bugreport

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// BugReport holds all data for composing a GitHub issue.
type BugReport struct {
	Title          string
	Description    string
	Steps          string
	Severity       string
	Category       string
	Environment    Environment
	IncludeEnv     bool
	LogExcerpt     string
	SessionInfo    string
	ExtLogExcerpt  string
}

// FormatIssueBody renders the bug report as a markdown issue body.
func (r *BugReport) FormatIssueBody() string {
	var b strings.Builder

	b.WriteString("## Bug Report\n\n")

	b.WriteString("### Description\n")
	b.WriteString(r.Description)
	b.WriteString("\n\n")

	if r.Steps != "" {
		b.WriteString("### Steps to Reproduce\n")
		b.WriteString(r.Steps)
		b.WriteString("\n\n")
	}

	if r.Severity != "" || r.Category != "" {
		b.WriteString("### Classification\n")
		if r.Severity != "" {
			fmt.Fprintf(&b, "- **Severity:** %s\n", r.Severity)
		}
		if r.Category != "" {
			fmt.Fprintf(&b, "- **Category:** %s\n", r.Category)
		}
		b.WriteString("\n")
	}

	if r.IncludeEnv {
		b.WriteString("### Environment\n")
		b.WriteString(r.Environment.FormatTable())
		b.WriteString("\n")
	}

	if r.LogExcerpt != "" {
		b.WriteString("<details>\n")
		if r.SessionInfo != "" {
			fmt.Fprintf(&b, "<summary>Log excerpt (%s)</summary>\n\n", r.SessionInfo)
		} else {
			b.WriteString("<summary>Log excerpt</summary>\n\n")
		}
		b.WriteString("```jsonl\n")
		b.WriteString(r.LogExcerpt)
		if !strings.HasSuffix(r.LogExcerpt, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
		b.WriteString("</details>\n\n")
	}

	if r.ExtLogExcerpt != "" {
		b.WriteString("<details>\n")
		b.WriteString("<summary>External tool logs</summary>\n\n")
		b.WriteString("```\n")
		b.WriteString(r.ExtLogExcerpt)
		if !strings.HasSuffix(r.ExtLogExcerpt, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
		b.WriteString("</details>\n\n")
	}

	fmt.Fprintf(&b, "---\n*Filed via `%s report bug`*\n", branding.Get().AppName)

	return b.String()
}
