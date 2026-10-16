package mcphealth

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

func TestBuildProcessEnv_ResolvesHomeVar(t *testing.T) {
	t.Parallel()

	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}

	env := map[string]string{
		"TEST_DIR": "${HOME}/subdir",
	}

	result := buildProcessEnv(env)

	expected := "TEST_DIR=" + home + "/subdir"
	if !slices.Contains(result, expected) {
		t.Errorf("expected %q in env, not found", expected)
	}
}

func TestBuildProcessEnv_MissingVarStaysLiteral(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"TOKEN": "${QSDEV_TEST_NONEXISTENT_VAR_ABCDEF}",
	}

	result := buildProcessEnv(env)

	expected := "TOKEN=${QSDEV_TEST_NONEXISTENT_VAR_ABCDEF}"
	if !slices.Contains(result, expected) {
		t.Errorf("expected %q in env (literal unresolved), not found", expected)
	}
}

func TestBuildProcessEnv_InheritsOSEnv(t *testing.T) {
	t.Parallel()

	result := buildProcessEnv(nil)

	if len(result) == 0 {
		t.Error("expected inherited OS environment entries, got empty slice")
	}
}

// TestStartServer_UsesDir is the U16-10 regression: a stdio server is started
// in cfg.Dir (the project root .mcp.json belongs to), as Claude Code starts
// it, not in qsdev's own working directory. An empty Dir keeps the process
// cwd, as on base. The helper server reports its cwd, so no shell is needed
// and the test runs on every OS.
func TestStartServer_UsesDir(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, dir, want string
	}{
		{name: "dir set", dir: t.TempDir()},
		{name: "empty dir keeps cwd", dir: "", want: cwd},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := fakeModeServerConfig(t, "cwd")
			cfg.Dir = tt.dir
			want := tt.want
			if want == "" {
				want = tt.dir
			}
			got := startedServerCwd(t, cfg)
			assertSameDir(t, got, want)
		})
	}
}

// startedServerCwd starts cfg (a "cwd"-mode helper server) and returns the
// working directory it reports in serverInfo.name.
func startedServerCwd(t *testing.T, cfg ServerConfig) string {
	t.Helper()
	proc, err := startServer(cfg)
	if err != nil {
		t.Fatalf("startServer: %v", err)
	}
	t.Cleanup(func() { _ = proc.Close() })

	result, err := proc.SendRequest(1, "initialize", initializeParams)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	var init struct {
		ServerInfo struct {
			Name string `json:"name"`
		} `json:"serverInfo"`
	}
	if err := json.Unmarshal(result, &init); err != nil {
		t.Fatalf("decoding initialize result %s: %v", result, err)
	}
	return init.ServerInfo.Name
}

// assertSameDir fails unless got and want name the same directory (compared
// by file identity, since a temp dir may be reached through a symlink).
func assertSameDir(t *testing.T, got, want string) {
	t.Helper()
	gotInfo, err := os.Stat(got)
	if err != nil {
		t.Fatalf("stat child cwd %q: %v", got, err)
	}
	wantInfo, err := os.Stat(want)
	if err != nil {
		t.Fatalf("stat %q: %v", want, err)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Errorf("child cwd = %q, want %q", got, want)
	}
}
