package render

import "io"

// Format enumerates supported output formats for PostureReport rendering.
type Format string

const (
	Text  Format = "text"
	JSON  Format = "json"
	SARIF Format = "sarif"
	Badge Format = "badge"
)

// Options configures output rendering of a PostureReport.
type Options struct {
	Verbose   bool
	Quiet     bool
	JSON      bool
	SARIF     bool
	Badge     bool
	BadgeType string
	Fix       bool
	UseColor  bool
	Writer    io.Writer
	Section   string
}
