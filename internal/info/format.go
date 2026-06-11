package info

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/timeutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// FormatDefault renders a human-readable multi-line summary of the project.
func FormatDefault(info *ProjectInfo, w io.Writer) error {
	fmt.Fprintf(w, "Project:       %s\n", info.ProjectName)
	if len(info.Ecosystems) > 0 {
		fmt.Fprintf(w, "Ecosystems:    %s\n", strings.Join(info.Ecosystems, ", "))
	}
	fmt.Fprintf(w, "Security:      %s\n", info.SecurityProfile)
	fmt.Fprintf(w, "%s Version:  %s\n", branding.Get().AppName, info.QsdevVersion)
	fmt.Fprintf(w, "Config:        v%d\n", info.ConfigVersion)
	fmt.Fprintf(w, "Managed Files: %d\n", info.ManagedFileCount)
	fmt.Fprintf(w, "Active Tools:  %d\n", info.ActiveToolCount)
	// Show category breakdown if any.
	for cat, count := range info.ToolsByCategory {
		fmt.Fprintf(w, "  %s: %d\n", cat, count)
	}
	if info.ClaudeCodeEnabled {
		fmt.Fprintf(w, "Claude Code:   enabled\n")
	}
	if !info.LastUpdated.IsZero() {
		fmt.Fprintf(w, "Last Updated:  %s\n", RelativeTime(info.LastUpdated))
	} else {
		fmt.Fprintf(w, "Last Updated:  never\n")
	}
	return nil
}

// FormatOneline renders a single-line summary suitable for prompts and scripts.
func FormatOneline(info *ProjectInfo, w io.Writer) error {
	ecos := "none"
	if len(info.Ecosystems) > 0 {
		ecos = strings.Join(info.Ecosystems, ",")
	}
	updated := "never"
	if !info.LastUpdated.IsZero() {
		updated = RelativeTime(info.LastUpdated)
	}
	fmt.Fprintf(w, "%s | %s | %s | %d tools | %s\n",
		info.ProjectName, ecos, info.SecurityProfile,
		info.ActiveToolCount, updated)
	return nil
}

// FormatJSON renders the project info as indented JSON.
func FormatJSON(info *ProjectInfo, w io.Writer) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(append(data, '\n'))
	return err
}

// RelativeTime formats a time.Time as a human-readable relative duration.
// It delegates to timeutil.RelativeTime for the shared implementation.
func RelativeTime(t time.Time) string {
	return timeutil.RelativeTime(t)
}
