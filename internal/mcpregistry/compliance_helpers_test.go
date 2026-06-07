package mcpregistry

import "testing"

func TestHasPlaintextSecrets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{
			name: "variable reference is not a secret",
			env:  map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
			want: false,
		},
		{
			name: "secret_ prefix flagged as secret",
			env:  map[string]string{"KEY": "secret_test_xxxxxxxxxxxxxxxxxxxx"},
			want: true,
		},
		{
			name: "empty env map",
			env:  nil,
			want: false,
		},
		{
			name: "short value is not a secret",
			env:  map[string]string{"DEBUG": "true"},
			want: false,
		},
		{
			name: "token_ prefix flagged as secret",
			env:  map[string]string{"GH": "token_ABCDEFGHIJKLMNOPQRSTUVWXYZab"},
			want: true,
		},
		{
			name: "mixed env with one secret",
			env: map[string]string{
				"SAFE":  "${SOME_VAR}",
				"OOPS":  "secret_myreallylongsecretvalue123",
				"DEBUG": "1",
			},
			want: true,
		},
		{
			name: "long path is not flagged",
			env:  map[string]string{"PATH": "/usr/local/share/docs"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			def := &McpServerDefinition{Env: tt.env}
			got := hasPlaintextSecrets(def)
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
		want    bool
	}{
		{name: "qsdev is local", command: "qsdev", want: true},
		{name: "npx is not local", command: "npx", want: false},
		{name: "absolute path is local", command: "/usr/bin/my-server", want: true},
		{name: "uvx is not local", command: "uvx", want: false},
		{name: "pipx is not local", command: "pipx", want: false},
		{name: "npm is not local", command: "npm", want: false},
		{name: "nix store path is local", command: "/nix/store/abc-server/bin/server", want: true},
		{name: "custom binary is local", command: "my-custom-server", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			def := &McpServerDefinition{Command: tt.command}
			got := isLocalOnly(def)
			if got != tt.want {
				t.Errorf("isLocalOnly() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasNpxDashY(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		args    []string
		want    bool
	}{
		{
			name:    "npx with -y flag",
			command: "npx",
			args:    []string{"-y", "@upstash/context7-mcp"},
			want:    true,
		},
		{
			name:    "npx with --yes flag",
			command: "npx",
			args:    []string{"--yes", "pkg"},
			want:    true,
		},
		{
			name:    "npx without -y flag",
			command: "npx",
			args:    []string{"@anthropic-ai/mcp-github"},
			want:    false,
		},
		{
			name:    "non-npx command with -y flag",
			command: "node",
			args:    []string{"-y"},
			want:    false,
		},
		{
			name:    "npx with empty args",
			command: "npx",
			args:    []string{},
			want:    false,
		},
		{
			name:    "npx with nil args",
			command: "npx",
			args:    nil,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			def := &McpServerDefinition{Command: tt.command, Args: tt.args}
			got := hasNpxDashY(def)
			if got != tt.want {
				t.Errorf("hasNpxDashY() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasVerifiedProvenance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{name: "nix store path", command: "/nix/store/abc123-pkg/bin/server", want: true},
		{name: "qsdev binary", command: "qsdev", want: true},
		{name: "npx has no provenance", command: "npx", want: false},
		{name: "uvx has no provenance", command: "uvx", want: false},
		{name: "usr local bin has no provenance", command: "/usr/local/bin/server", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			def := &McpServerDefinition{Command: tt.command}
			got := hasVerifiedProvenance(def)
			if got != tt.want {
				t.Errorf("hasVerifiedProvenance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasExternalAttestation(t *testing.T) {
	t.Parallel()

	def := &McpServerDefinition{Command: "anything"}
	if hasExternalAttestation(def) {
		t.Error("hasExternalAttestation() = true, want false (placeholder)")
	}
}
