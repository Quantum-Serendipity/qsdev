package devenv

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
)

// envRedactor is the shared value-level secret scrubber, built once (it compiles
// a set of regexes). A variable can pass the name filter yet still carry a secret
// in its VALUE — e.g. DATABASE_URL=postgres://user:pass@host or a token-shaped
// value in a generically-named var — so every emitted value is run through it to
// strip URL userinfo and secret-shaped patterns.
var envRedactor = sync.OnceValue(logging.NewRedactor)

// envInfo probes the development environment: PATH composition, listening TCP
// ports, the qsdev-managed tool catalog, and a filtered view of the process
// environment. It never emits the values of sensitive environment variables.
type envInfo struct {
	projectRoot string
}

func newEnvInfo(projectRoot string) *envInfo { return &envInfo{projectRoot: projectRoot} }

// sensitiveEnvPrefixes identify cloud-provider environment variable namespaces
// whose values must never be emitted, even when the specific name carries no
// credential keyword (e.g. GCP_PROJECT, GOOGLE_CLOUD_PROJECT). This source-side
// filtering is defense-in-depth alongside the ContentSafety middleware, which
// also redacts secret-shaped values from results.
var sensitiveEnvPrefixes = []string{"AWS_", "AZURE_", "GCP_", "GOOGLE_", "GH_", "GITHUB_"}

// isSensitiveEnv reports whether the named variable's value must be withheld.
// The keyword match delegates to the shared secrets.MatchesSensitiveKeyPattern
// predicate (token-boundary: password/secret/token/session/passwd/pwd/access_key/
// key/…), so this probe, the log redactor, and the external-log scrubber share
// one authority. The pattern predicate — rather than IsSensitiveName — is used so
// a connection var whose credential lives in its VALUE (e.g. DATABASE_URL) is
// still value-scrubbed (host preserved) rather than withheld wholesale.
func isSensitiveEnv(name string) bool {
	if secrets.MatchesSensitiveKeyPattern(name) {
		return true
	}
	up := strings.ToUpper(name)
	for _, p := range sensitiveEnvPrefixes {
		if strings.HasPrefix(up, p) {
			return true
		}
	}
	return false
}

// handle runs the requested probe (or all of them) and returns a structured
// environment summary.
func (e *envInfo) handle(_ context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	probe := strings.ToLower(toolutil.StringArgOr(req.Arguments, "probe", "all"))

	result := map[string]any{}
	switch probe {
	case "path":
		result["path"] = e.probePath()
	case "ports":
		result["ports"] = probePorts()
	case "tools":
		result["tools"] = e.probeTools()
	case "env":
		result["env"] = probeEnv()
	case "all", "":
		result["path"] = e.probePath()
		result["ports"] = probePorts()
		result["tools"] = e.probeTools()
		result["env"] = probeEnv()
	default:
		return toolutil.NotConfigured("unknown probe",
			map[string]any{"got": probe, "allowed": []string{"path", "ports", "tools", "env", "all"}}), nil
	}

	text := fmt.Sprintf("env_info: probe=%s on %s", probe, runtime.GOOS)
	return toolutil.Result(text, result), nil
}

// pathEntry is one PATH component tagged with its origin category.
type pathEntry struct {
	Path     string `json:"path"`
	Category string `json:"category"` // system | user | nix | tool-managed
}

// probePath splits PATH and categorizes each entry.
func (e *envInfo) probePath() map[string]any {
	home, _ := os.UserHomeDir()
	raw := os.Getenv("PATH")
	parts := strings.Split(raw, string(os.PathListSeparator))

	entries := make([]pathEntry, 0, len(parts))
	counts := map[string]int{}
	for _, p := range parts {
		if p == "" {
			continue
		}
		cat := categorizePathEntry(p, home)
		counts[cat]++
		entries = append(entries, pathEntry{Path: p, Category: cat})
	}
	return map[string]any{"entries": entries, "count": len(entries), "by_category": counts}
}

// categorizePathEntry classifies a PATH component by its provenance.
func categorizePathEntry(entry, home string) string {
	switch {
	case strings.HasPrefix(entry, "/nix/"):
		return "nix"
	case strings.Contains(entry, "/.devenv/"),
		strings.Contains(entry, "/.cargo/"),
		strings.Contains(entry, "/.rustup/"),
		strings.Contains(entry, "/go/bin"),
		strings.Contains(entry, "node_modules/.bin"),
		strings.Contains(entry, "/.local/share/"):
		return "tool-managed"
	case home != "" && strings.HasPrefix(entry, home):
		return "user"
	default:
		return "system"
	}
}

// probePorts lists locally listening TCP ports. It is Linux-specific (reads
// /proc/net/tcp{,6}); on other platforms it returns an unsupported marker rather
// than failing.
func probePorts() map[string]any {
	if runtime.GOOS != "linux" {
		return map[string]any{"supported": false, "note": "port probing requires Linux /proc/net/tcp"}
	}
	portSet := map[int]bool{}
	for _, f := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		readListeningPorts(f, portSet)
	}
	ports := make([]int, 0, len(portSet))
	for p := range portSet {
		ports = append(ports, p)
	}
	sort.Ints(ports)
	return map[string]any{"supported": true, "listening": ports, "count": len(ports)}
}

// readListeningPorts parses a /proc/net/tcp-format file, recording the local
// port of every socket in the LISTEN (0x0A) state into out.
func readListeningPorts(path string, out map[int]bool) {
	f, err := os.Open(path) //nolint:gosec // fixed /proc path, not user input
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	first := true
	for sc.Scan() {
		if first { // skip the header row
			first = false
			continue
		}
		fields := strings.Fields(sc.Text())
		if len(fields) < 4 || fields[3] != "0A" {
			continue
		}
		local := fields[1]
		colon := strings.LastIndex(local, ":")
		if colon < 0 {
			continue
		}
		if port, err := strconv.ParseInt(local[colon+1:], 16, 32); err == nil {
			out[int(port)] = true
		}
	}
}

// probeTools delegates to the qsdev tool registry, listing the managed tool
// catalog (name + category) and whether each tool's name resolves to a binary on
// PATH. It does not execute any binary, so it stays fast and side-effect free.
func (e *envInfo) probeTools() map[string]any {
	reg, err := toolreg.Default()
	if err != nil {
		return map[string]any{"available": false, "error": err.Error()}
	}
	all := reg.All()
	tools := make([]map[string]any, 0, len(all))
	for _, t := range all {
		onPath := false
		if _, err := exec.LookPath(t.Name); err == nil {
			onPath = true
		}
		tools = append(tools, map[string]any{
			"name":     t.Name,
			"category": string(t.Category),
			"on_path":  onPath,
		})
	}
	return map[string]any{"available": true, "count": len(tools), "tools": tools}
}

// probeEnv returns a filtered snapshot of the process environment: the values of
// non-sensitive variables (each run through the value-level secret scrubber so an
// embedded credential — e.g. user:pass@ in a DSN — is stripped), plus the names
// (never values) of variables whose names match a sensitive pattern.
func probeEnv() map[string]any {
	safe := map[string]string{}
	var filtered []string
	r := envRedactor()
	for _, kv := range os.Environ() {
		name, value, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if isSensitiveEnv(name) {
			filtered = append(filtered, name)
			continue
		}
		safe[name] = r.RedactString(value)
	}
	sort.Strings(filtered)
	return map[string]any{
		"variables":      safe,
		"filtered_names": filtered,
		"total":          len(safe) + len(filtered),
		"filtered_count": len(filtered),
	}
}
