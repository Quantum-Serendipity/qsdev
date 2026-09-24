package mcpregistry

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestHasPlaintextSecrets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		def  McpServerDefinition
		want bool
	}{
		{
			name: "variable reference is not a secret",
			def:  McpServerDefinition{Env: map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"}},
			want: false,
		},
		{
			name: "secret_ prefix flagged as secret",
			def:  McpServerDefinition{Env: map[string]string{"KEY": "secret_test_xxxxxxxxxxxxxxxxxxxx"}},
			want: true,
		},
		{
			name: "empty env map",
			def:  McpServerDefinition{},
			want: false,
		},
		{
			name: "short value is not a secret",
			def:  McpServerDefinition{Env: map[string]string{"DEBUG": "true"}},
			want: false,
		},
		{
			name: "token_ prefix flagged as secret",
			def:  McpServerDefinition{Env: map[string]string{"GH": "token_ABCDEFGHIJKLMNOPQRSTUVWXYZab"}},
			want: true,
		},
		{
			name: "mixed env with one secret",
			def: McpServerDefinition{Env: map[string]string{
				"SAFE":  "${SOME_VAR}",
				"OOPS":  "secret_myreallylongsecretvalue123",
				"DEBUG": "1",
			}},
			want: true,
		},
		{
			name: "long path is not flagged",
			def:  McpServerDefinition{Env: map[string]string{"PATH": "/usr/local/share/docs"}},
			want: false,
		},
		{
			name: "long absolute path with mixed case and digits is not flagged",
			def:  McpServerDefinition{Env: map[string]string{"DATA": "/home/User1/projects/Docs2026/offline/zim/archive"}},
			want: false,
		},
		{
			name: "40-char lowercase hex digest is not flagged",
			def:  McpServerDefinition{Env: map[string]string{"REV": "0123456789abcdef0123456789abcdef01234567"}},
			want: false,
		},
		{
			name: "non-secret config values are not flagged",
			def:  McpServerDefinition{Env: map[string]string{"AWS_DEFAULT_REGION": "us-east-1", "TOKEN_FILE": "/run/secrets/token"}},
			want: false,
		},
		{
			name: "AWS secret access key with slashes flagged",
			def:  McpServerDefinition{Env: map[string]string{"AWS_SECRET": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"}},
			want: true,
		},
		{
			name: "AWS access key id flagged",
			def:  McpServerDefinition{Env: map[string]string{"AWS_ACCESS_KEY_ID": "AKIAIOSFODNN7EXAMPLE"}},
			want: true,
		},
		{
			name: "slack bot token flagged",
			def:  McpServerDefinition{Env: map[string]string{"SLACK": "xoxb-1234567890-abcdefghij"}},
			want: true,
		},
		{
			name: "gitlab token flagged",
			def:  McpServerDefinition{Env: map[string]string{"GL": "glpat-abcdefghijklmnopqrst"}},
			want: true,
		},
		{
			name: "literal password under a credential key flagged",
			def:  McpServerDefinition{Env: map[string]string{"DB_PASSWORD": "hunter2hunter2"}},
			want: true,
		},
		{
			name: "literal next to a variable reference flagged",
			def:  McpServerDefinition{Env: map[string]string{"X": "${HOME}ghp_abcdefghijklmnopqrstuvwxyz0123456789"}},
			want: true,
		},
		{
			name: "secret-shaped arg flagged",
			def:  McpServerDefinition{Args: []string{"--api-key", "sk-ant-api03-abcdefghijklmnopqrstuvwxyz"}},
			want: true,
		},
		{
			name: "literal value of a credential flag flagged",
			def:  McpServerDefinition{Args: []string{"--token", "abcd1234efgh"}},
			want: true,
		},
		{
			name: "credential flag with equals value flagged",
			def:  McpServerDefinition{Args: []string{"--password=correcthorse"}},
			want: true,
		},
		{
			name: "credential switch followed by another flag not flagged",
			def:  McpServerDefinition{Args: []string{"--no-token", "--verbose-logging"}},
			want: false,
		},
		{
			name: "credential flag with variable reference not flagged",
			def:  McpServerDefinition{Args: []string{"--token", "${GITHUB_TOKEN}"}},
			want: false,
		},
		{
			name: "plain args not flagged",
			def:  McpServerDefinition{Args: []string{"-y", "@modelcontextprotocol/server-filesystem", "/home/user/project"}},
			want: false,
		},
		{
			name: "URL with credential query parameter flagged",
			def:  McpServerDefinition{URL: "https://mcp.example.com/sse?api_key=abcdef0123456789"},
			want: true,
		},
		{
			name: "URL with userinfo flagged",
			def:  McpServerDefinition{URL: "https://user:s3cretpass@mcp.example.com/mcp"},
			want: true,
		},
		{
			name: "plain URL not flagged",
			def:  McpServerDefinition{URL: "https://mcp.socket.dev/"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := hasPlaintextSecrets(&tt.def)
			if got != tt.want {
				t.Errorf("hasPlaintextSecrets() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsLocalOnly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		args    []string
		want    bool
	}{
		{name: "qsdev is local", command: "qsdev", want: true},
		{name: "npx is not local", command: "npx", want: false},
		{name: "absolute path is local", command: "/usr/bin/my-server", want: true},
		{name: "uvx is not local", command: "uvx", want: false},
		{name: "pipx is not local", command: "pipx", want: false},
		{name: "npm is not local", command: "npm", want: false},
		{name: "pnpx is not local", command: "pnpx", want: false},
		{name: "bunx is not local", command: "bunx", want: false},
		{name: "absolute path to npx is not local", command: "/usr/bin/npx", want: false},
		{name: "nix store uvx is not local", command: "/nix/store/abc-uv/bin/uvx", want: false},
		{name: "windows npx shim is not local", command: `C:\nodejs\npx.cmd`, want: false},
		{name: "windows uvx exe is not local", command: "uvx.exe", want: false},
		{name: "nix store path is local", command: "/nix/store/abc-server/bin/server", want: true},
		{name: "custom binary is local", command: "my-custom-server", want: true},
		{name: "absolute npx is not local", command: "/usr/bin/npx", args: []string{"-y", "x"}, want: false},
		{name: "nixos npx is not local", command: "/run/current-system/sw/bin/npx", args: []string{"x"}, want: false},
		{name: "windows npx shim is not local", command: `C:\Program Files\nodejs\npx.cmd`, args: []string{"x"}, want: false},
		{name: "bunx is not local", command: "bunx", args: []string{"x"}, want: false},
		{name: "pnpx is not local", command: "pnpx", args: []string{"x"}, want: false},
		{name: "pnpm dlx is not local", command: "pnpm", args: []string{"dlx", "x"}, want: false},
		{name: "yarn dlx is not local", command: "yarn", args: []string{"dlx", "x"}, want: false},
		{name: "bun x is not local", command: "bun", args: []string{"x", "pkg"}, want: false},
		{name: "uv tool run is not local", command: "uv", args: []string{"tool", "run", "pkg"}, want: false},
		{name: "deno is not local", command: "deno", args: []string{"run", "https://example.com/srv.ts"}, want: false},
		{name: "docker run is not local", command: "docker", args: []string{"run", "-i", "img"}, want: false},
		{name: "env-wrapped npx is not local", command: "env", args: []string{"FOO=bar", "npx", "x"}, want: false},
		{name: "env split-string wrapper fails closed", command: "env", args: []string{"-S", "npx -y pkg"}, want: false},
		{name: "env split-string long form fails closed", command: "/usr/bin/env", args: []string{"--split-string=npx -y pkg"}, want: false},
		{name: "env clustered split-string fails closed", command: "env", args: []string{"-iS", "npx -y pkg"}, want: false},
		{name: "yarn create is not local", command: "yarn", args: []string{"create", "x"}, want: false},
		{name: "shell wrapper fails closed", command: "sh", args: []string{"-c", "exec server"}, want: false},
		{name: "bash wrapper fails closed", command: "/bin/bash", args: []string{"-lc", "server"}, want: false},
		{name: "pnpm exec of local bin is local", command: "pnpm", args: []string{"exec", "my-server"}, want: true},
		{name: "bun running a local file is local", command: "bun", args: []string{"server.js"}, want: true},
		{name: "env-wrapped local binary is local", command: "env", args: []string{"FOO=bar", "/usr/bin/srv"}, want: true},
		{name: "offline npx is local", command: "npx", args: []string{"--offline", "pkg@1.0.0"}, want: true},
		{name: "docker with pull never is local", command: "docker", args: []string{"run", "--pull", "never", "img"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			def := &McpServerDefinition{Command: tt.command, Args: tt.args}
			got := isLocalOnly(def)
			if got != tt.want {
				t.Errorf("isLocalOnly(%q %v) = %v, want %v", tt.command, tt.args, got, tt.want)
			}
		})
	}
}

func TestHasRuntimeAutoInstall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		args    []string
		want    bool
	}{
		{name: "npx with -y", command: "npx", args: []string{"-y", "@upstash/context7-mcp"}, want: true},
		{name: "npx with --yes", command: "npx", args: []string{"--yes", "pkg"}, want: true},
		// npx assumes --yes when stdin is not a TTY, as for every stdio server.
		{name: "npx without -y", command: "npx", args: []string{"@anthropic-ai/mcp-github"}, want: true},
		{name: "npx pinned but online", command: "npx", args: []string{"pkg@1.2.3"}, want: true},
		{name: "npx offline but unpinned", command: "npx", args: []string{"--no", "pkg"}, want: true},
		{name: "npx offline with dist-tag", command: "npx", args: []string{"--offline", "pkg@latest"}, want: true},
		{name: "npx offline and pinned", command: "npx", args: []string{"--no", "pkg@1.2.3"}, want: false},
		{name: "npx offline scoped and pinned", command: "npx", args: []string{"--offline", "@scope/pkg@1.2.3"}, want: false},
		{name: "npx offline package flag pinned", command: "npx", args: []string{"--offline", "-p", "pkg@2.0.0", "bin"}, want: false},
		{name: "path-qualified npx", command: "/usr/bin/npx", args: []string{"pkg"}, want: true},
		{name: "uvx offline pinned", command: "uvx", args: []string{"--offline", "mcp-server-fetch==1.2.3"}, want: false},
		{name: "uvx unpinned", command: "uvx", args: []string{"mcp-server-fetch"}, want: true},
		{name: "docker digest-pinned offline", command: "docker", args: []string{"run", "--pull=never", "img@sha256:" + strings.Repeat("a", 64)}, want: false},
		{name: "docker tag", command: "docker", args: []string{"run", "img:latest"}, want: true},
		{name: "shell wrapper fails closed", command: "bash", args: []string{"-c", "srv"}, want: true},
		{name: "non-launcher with -y flag", command: "node", args: []string{"-y"}, want: false},
		{name: "local binary", command: "/usr/local/bin/srv", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			def := &McpServerDefinition{Command: tt.command, Args: tt.args}
			got := hasRuntimeAutoInstall(def)
			if got != tt.want {
				t.Errorf("hasRuntimeAutoInstall(%q %v) = %v, want %v", tt.command, tt.args, got, tt.want)
			}
		})
	}
}

func TestProvenanceResolverVerified(t *testing.T) {
	t.Parallel()

	const (
		storeBin = "/nix/store/0123456789abcdfghijklmnpqrsvwxyz-pkg/bin/server"
		self     = "/nix/store/zyxwvsrqpnmlkjihgfdcba9876543210-qsdev/bin/qsdev"
	)
	errMissing := errors.New("no such file")

	tests := []struct {
		name     string
		command  string
		lookPath string // result of looking up "qsdev" on PATH; empty means not found
		symlinks map[string]string
		missing  []string
		want     bool
	}{
		{name: "nix store path", command: storeBin, want: true},
		{name: "store object itself", command: "/nix/store/0123456789abcdfghijklmnpqrsvwxyz-script", want: true},
		{name: "traversal out of the store", command: "/nix/store/../../tmp/evil", want: false},
		{name: "traversal inside a store path", command: "/nix/store/0123456789abcdfghijklmnpqrsvwxyz-pkg/../../../tmp/evil", want: false},
		{name: "malformed store hash", command: "/nix/store/abc123-pkg/bin/server", want: false},
		{name: "hash with letters outside nix base32", command: "/nix/store/0123456789abcdefghijklmnopqrstuv-pkg/bin/x", want: false},
		{name: "duplicate separators", command: "/nix/store//0123456789abcdfghijklmnpqrsvwxyz-pkg/bin/x", want: false},
		{name: "missing store path", command: storeBin, missing: []string{storeBin}, want: false},
		{name: "store symlink escaping the store", command: storeBin, symlinks: map[string]string{storeBin: "/home/u/evil"}, want: false},
		{name: "store symlink to another store path", command: storeBin, symlinks: map[string]string{storeBin: self}, want: true},
		{name: "qsdev resolving to the running binary", command: "qsdev", lookPath: self, want: true},
		{
			name:     "qsdev via profile symlink",
			command:  "qsdev",
			lookPath: "/run/current-system/sw/bin/qsdev",
			symlinks: map[string]string{"/run/current-system/sw/bin/qsdev": self},
			want:     true,
		},
		{name: "planted qsdev earlier on PATH", command: "qsdev", lookPath: "/home/u/proj/.bin/qsdev", want: false},
		{name: "qsdev not on PATH", command: "qsdev", want: false},
		{name: "npx has no provenance", command: "npx", lookPath: self, want: false},
		{name: "usr local bin has no provenance", command: "/usr/local/bin/server", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := provenanceResolver{
				lookPath: func(string) (string, error) {
					if tt.lookPath == "" {
						return "", errMissing
					}
					return tt.lookPath, nil
				},
				evalSymlinks: func(p string) (string, error) {
					if slices.Contains(tt.missing, p) {
						return "", errMissing
					}
					if target, ok := tt.symlinks[p]; ok {
						return target, nil
					}
					return p, nil
				},
				executable: func() (string, error) { return self, nil },
			}
			if got := r.verified(tt.command); got != tt.want {
				t.Errorf("verified(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestHasExternalAttestation(t *testing.T) {
	t.Parallel()

	def := &McpServerDefinition{Command: "anything"}
	if hasExternalAttestation(def) {
		t.Error("hasExternalAttestation() = true, want false (default no-op checker)")
	}
}
