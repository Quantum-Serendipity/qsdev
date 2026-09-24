package doctor

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
)

// MCP server statuses in the doctor report.
const (
	MCPStatusOK            = "ok"
	MCPStatusDegraded      = "degraded"
	MCPStatusMisconfigured = "misconfigured"
)

// MCP server transports in the doctor report.
const (
	mcpTransportStdio = "stdio"
	mcpTransportHTTP  = "http"
)

// emptyServerName is the name mcphealth.ValidateConfig reports for a server
// entry with an empty name.
const emptyServerName = "(empty)"

// NewMCPSection builds the doctor's MCP section from the configured servers
// and the findings of mcphealth.ValidateConfig for them. It never starts a
// server: an .mcp.json in an untrusted checkout names commands the doctor must
// not run. Findings are attached to the server they name, in order; a server
// with an error is misconfigured and one with only warnings is degraded.
// warnings carry inputs that could not be read. It returns nil when there is
// nothing to report.
func NewMCPSection(servers []mcphealth.ServerConfig, findings []mcphealth.ConfigWarning, warnings []string) *MCPSection {
	if len(servers) == 0 && len(warnings) == 0 {
		return nil
	}
	byServer := make(map[string][]mcphealth.ConfigWarning, len(findings))
	for _, f := range findings {
		byServer[f.Server] = append(byServer[f.Server], f)
	}

	ms := &MCPSection{Detected: true, Servers: []MCPServerInfo{}, Warnings: warnings}
	for _, srv := range servers {
		name := srv.Name
		if name == "" {
			name = emptyServerName
		}
		info := MCPServerInfo{Name: name, Transport: mcpTransportStdio, Status: MCPStatusOK}
		if srv.URL != "" {
			info.Transport = mcpTransportHTTP
		}
		for _, f := range byServer[name] {
			info.Issues = append(info.Issues, MCPIssue{
				Severity:    f.Severity,
				Message:     f.Message,
				Remediation: f.Remediation,
			})
			info.Status = worseMCPStatus(info.Status, f.Severity)
		}
		ms.Servers = append(ms.Servers, info)
	}
	return ms
}

// worseMCPStatus returns the server status after a finding of severity.
// An unknown severity counts as an error so it is never reported as healthy.
func worseMCPStatus(current, severity string) string {
	if severity == mcphealth.SeverityWarning {
		if current == MCPStatusOK {
			return MCPStatusDegraded
		}
		return current
	}
	return MCPStatusMisconfigured
}

// displayMCPServerName renders a server name from .mcp.json for the terminal.
// The name is repository content, so one holding a control, format or other
// non-printable character (an ANSI escape sequence could rewrite or hide the
// lines around it) is shown quoted with those characters escaped.
func displayMCPServerName(name string) string {
	if strings.IndexFunc(name, func(r rune) bool { return !unicode.IsPrint(r) }) >= 0 {
		return strconv.Quote(name)
	}
	return name
}
