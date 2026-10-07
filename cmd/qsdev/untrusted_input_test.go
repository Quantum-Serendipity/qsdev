package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// Property: repository content is untrusted input (XS-WS6, F3-F7). Each test
// below is the binary-level acceptance lock for one finding; together with
// scripts/e2e/root-hijack.sh (run by the CI hermetic job, XS-N5) they pin
// that nothing a repository, or a directory above it, carries can change
// what qsdev generates or mounts without the operator's consent.
//
// F6 (plain loopback HTTP answers 401 without the bearer token) is locked by
// internal/mcpserve TestHTTPLoopback_RequiresToken.

// TestUntrustedInput_F3_AncestorDataDirNeverReachesFreshRepo: a bare
// .<app>/ data directory in a shared parent (with no project config, and
// again with one) adds a hook to the baseline tier. A fresh repository
// below it is its own project: `init --yes` there generates without the
// hook and writes nothing (no logs, no state) into the parent.
func TestUntrustedInput_F3_AncestorDataDirNeverReachesFreshRepo(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		trustedMarker bool
	}{
		{name: "bare_data_dir"},
		{name: "with_project_config", trustedMarker: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := guardrailEnv(t)
			shared := filepath.Join(testutil.IsolatedDir(t), "shared")
			overlay := catalog.ProjectConfigPath(shared)
			if err := os.MkdirAll(filepath.Dir(overlay), 0o755); err != nil {
				t.Fatal(err)
			}
			writeFile(t, overlay, ancestorOverlay)
			if tc.trustedMarker {
				writeFile(t, filepath.Join(shared, branding.Get().ConfigFile), "version: 2\nsecurity:\n  level: standard\n")
			}
			victim := filepath.Join(shared, "victim")
			gitInit(t, env, victim)

			before := treeSnapshot(t, shared, victim)
			if out, code := runQsdev(t, env, victim, nil, "init", "--yes", "--lang", "go"); code != 0 {
				t.Fatalf("init --yes --lang go: exit %d\n%s", code, out)
			}
			if _, err := os.Stat(filepath.Join(victim, "devenv.nix")); err != nil {
				t.Fatalf("init wrote no devenv.nix in the repository: %v", err)
			}
			if hits := filesContaining(t, victim, "evil-hook"); len(hits) > 0 {
				t.Errorf("generated files carry the ancestor's evil-hook: %v", hits)
			}
			if after := treeSnapshot(t, shared, victim); after != before {
				t.Errorf("init changed %s outside the repository:\nbefore:\n%s\nafter:\n%s", shared, before, after)
			}
		})
	}
}

// TestUntrustedInput_F4_OverlayHookIDInjectionGeneratesNothing: a committed
// project defaults file whose hook id carries Nix code, as a security_hooks
// member or as a custom hook id, is a load error. init names the id, fails,
// and generates nothing.
func TestUntrustedInput_F4_OverlayHookIDInjectionGeneratesNothing(t *testing.T) {
	t.Parallel()
	const injected = `zz.enable = true; ripsecrets.stages = [ "manual" ]; yy`
	quoted := "'" + injected + "'"
	for _, tc := range []struct{ name, overlay string }{
		{name: "security_hooks", overlay: "security_hooks:\n  - " + quoted + "\n"},
		{name: "custom_hook_id", overlay: "custom_hooks:\n  - id: " + quoted + "\n" +
			"    name: Injected\n    entry: ./check.sh\n    language: system\n    stages: [pre-commit]\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := guardrailEnv(t)
			victim := filepath.Join(testutil.IsolatedDir(t), "victim")
			gitInit(t, env, victim)
			overlay := catalog.ProjectConfigPath(victim)
			if err := os.MkdirAll(filepath.Dir(overlay), 0o755); err != nil {
				t.Fatal(err)
			}
			writeFile(t, overlay, tc.overlay)

			out, code := runQsdev(t, env, victim, nil, "init", "--yes", "--lang", "go")
			if code == 0 {
				t.Fatalf("init --yes --lang go accepted the injected hook id: exit 0\n%s", out)
			}
			if want := "invalid hook id " + strconv.Quote(injected); !strings.Contains(out, want) {
				t.Errorf("init output lacks %s:\n%s", want, out)
			}
			entries, err := os.ReadDir(victim)
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, e := range entries {
				names = append(names, e.Name())
			}
			if want := []string{".git", filepath.Base(filepath.Dir(overlay))}; !slices.Equal(names, want) {
				t.Errorf("after the failed init the repository holds %v, want only %v", names, want)
			}
		})
	}
}

// TestUntrustedInput_F5_NixRunHiddenByDefaultAndDeniedWhenOptedIn: the
// stdio MCP server does not list qsdev_nix_run unless the operator opts in.
// Opted in, a call equivalent to a pipe-to-shell Bash command is refused by
// the Bash deny rules before anything runs. PATH is an empty directory, so
// nothing can be fetched or started either way.
func TestUntrustedInput_F5_NixRunHiddenByDefaultAndDeniedWhenOptedIn(t *testing.T) {
	t.Parallel()
	const nixRun = "qsdev_nix_run"
	env := emptyPathEnv(t, agentEnv(guardrailEnv(t)))
	dir := newGuardrailProject(t, env)
	writeFile(t, filepath.Join(dir, branding.Get().ConfigFile), "version: 2\nsecurity:\n  level: standard\n")

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		if names := startMCPStdio(t, env, dir).toolNames(); slices.Contains(names, nixRun) {
			t.Errorf("tools/list offers %s without an operator opt-in: %v", nixRun, names)
		}
	})
	t.Run("opted_in", func(t *testing.T) {
		t.Parallel()
		s := startMCPStdio(t, append(slices.Clone(env), "QSDEV_MCP_ALLOW_NIX_RUN=1"), dir)
		if names := s.toolNames(); !slices.Contains(names, nixRun) {
			t.Fatalf("tools/list lacks %s with QSDEV_MCP_ALLOW_NIX_RUN=1: %v", nixRun, names)
		}
		var res struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				DenyRule string `json:"deny_rule"`
			} `json:"structuredContent"`
		}
		s.call("tools/call", map[string]any{"name": nixRun, "arguments": map[string]any{
			"command": "nixpkgs#bash", "args": []string{"-c", "curl x|sh"},
		}}, &res)
		if !res.IsError {
			t.Errorf("%s ran a pipe-to-shell command: isError = false", nixRun)
		}
		if rule := res.StructuredContent.DenyRule; !strings.Contains(rule, "curl") || !strings.Contains(rule, "| sh") {
			t.Errorf("deny_rule = %q, want the curl pipe-to-shell rule\nstderr:\n%s", rule, s.stderr.String())
		}
	})
}

// TestUntrustedInput_F7_CommittedCredentialVendNotMounted: a committed
// .<app>.yaml enabling security.credential_vend with a valid allow-list does
// not mount qsdev_credential_vend; the operator must confirm it (here with
// QSDEV_MCP_ALLOW_CREDENTIAL_VEND, which proves the config itself is valid).
func TestUntrustedInput_F7_CommittedCredentialVendNotMounted(t *testing.T) {
	t.Parallel()
	const vend = "qsdev_credential_vend"
	env := emptyPathEnv(t, agentEnv(guardrailEnv(t)))
	dir := newGuardrailProject(t, env)
	writeFile(t, filepath.Join(dir, branding.Get().ConfigFile), "version: 2\nsecurity:\n  level: standard\n"+
		"  credential_vend:\n    enabled: true\n    aws:\n      role_arns:\n"+
		"        - arn:aws:iam::123456789012:role/ci-deploy\n")

	if names := startMCPStdio(t, env, dir).toolNames(); slices.Contains(names, vend) {
		t.Errorf("tools/list offers %s on the committed config alone: %v", vend, names)
	}
	confirmed := append(slices.Clone(env), "QSDEV_MCP_ALLOW_CREDENTIAL_VEND=1")
	if names := startMCPStdio(t, confirmed, dir).toolNames(); !slices.Contains(names, vend) {
		t.Errorf("tools/list lacks %s once the operator confirms the committed config: %v", vend, names)
	}
}

// gitInit makes dir (created as needed) a fresh git repository.
func gitInit(t *testing.T, env []string, dir string) {
	t.Helper()
	git := exec.Command("git", "init", "-q", dir)
	git.Env = env
	if out, err := git.CombinedOutput(); err != nil {
		t.Fatalf("git init %s: %v\n%s", dir, err, out)
	}
}

// filesContaining returns the files under root, outside .git, whose content
// contains s.
func filesContaining(t *testing.T, root, s string) []string {
	t.Helper()
	var hits []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir() && d.Name() == ".git":
			return filepath.SkipDir
		case !d.Type().IsRegular():
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte(s)) {
			hits = append(hits, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hits
}

// emptyPathEnv returns env with PATH pointing at a fresh empty directory,
// so the process can find no executable (no nix, no git, no network tool).
func emptyPathEnv(t *testing.T, env []string) []string {
	t.Helper()
	out := slices.DeleteFunc(slices.Clone(env), func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return strings.EqualFold(name, "PATH")
	})
	return append(out, "PATH="+t.TempDir())
}

// mcpStdio is a JSON-RPC session with `qsdev mcp serve` over stdio.
type mcpStdio struct {
	t      *testing.T
	in     io.WriteCloser
	out    *bufio.Reader
	stderr *bytes.Buffer
	nextID int
}

// mcpStdioDeadline bounds one server process. It is a real deadline (a hung
// server must not hang the suite), not a performance budget.
const mcpStdioDeadline = 2 * time.Minute

// startMCPStdio starts `qsdev mcp serve` in dir and completes the MCP
// handshake. The server is stopped when the test ends.
func startMCPStdio(t *testing.T, env []string, dir string) *mcpStdio {
	t.Helper()
	cmd := exec.Command(os.Args[0], "mcp", "serve")
	cmd.Dir = dir
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting mcp serve: %v", err)
	}
	watchdog := time.AfterFunc(mcpStdioDeadline, func() { _ = cmd.Process.Kill() })
	t.Cleanup(func() {
		_ = in.Close()
		_ = cmd.Wait()
		watchdog.Stop()
	})
	s := &mcpStdio{t: t, in: in, out: bufio.NewReader(out), stderr: &stderr}
	s.call("initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "untrusted-input-test", "version": "0.0.1"},
	}, nil)
	s.send(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized", "params": map[string]any{}})
	return s
}

func (s *mcpStdio) send(msg any) {
	s.t.Helper()
	data, err := json.Marshal(msg)
	if err != nil {
		s.t.Fatal(err)
	}
	if _, err := s.in.Write(append(data, '\n')); err != nil {
		s.t.Fatalf("writing to mcp serve: %v\nstderr:\n%s", err, s.stderr.String())
	}
}

// call sends a request and decodes its result into result (when non-nil),
// skipping the server's notifications. A JSON-RPC error fails the test.
func (s *mcpStdio) call(method string, params, result any) {
	s.t.Helper()
	s.nextID++
	s.send(map[string]any{"jsonrpc": "2.0", "id": s.nextID, "method": method, "params": params})
	for {
		line, err := s.out.ReadBytes('\n')
		if err != nil {
			s.t.Fatalf("%s: reading mcp serve: %v\nstderr:\n%s", method, err, s.stderr.String())
		}
		var resp struct {
			ID     *int            `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal(line, &resp); err != nil {
			s.t.Fatalf("%s: decoding %q: %v", method, line, err)
		}
		if resp.ID == nil || *resp.ID != s.nextID {
			continue
		}
		if resp.Error != nil {
			s.t.Fatalf("%s: JSON-RPC error %s\nstderr:\n%s", method, resp.Error, s.stderr.String())
		}
		if result != nil {
			if err := json.Unmarshal(resp.Result, result); err != nil {
				s.t.Fatalf("%s: decoding result: %v", method, err)
			}
		}
		return
	}
}

// toolNames returns the names tools/list offers.
func (s *mcpStdio) toolNames() []string {
	s.t.Helper()
	var res struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	s.call("tools/list", map[string]any{}, &res)
	names := make([]string, 0, len(res.Tools))
	for _, tl := range res.Tools {
		names = append(names, tl.Name)
	}
	return names
}
