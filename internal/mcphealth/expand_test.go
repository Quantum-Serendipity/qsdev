package mcphealth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

func fakeLookup(vars map[string]string) envLookup {
	return func(name string) (string, bool) {
		v, ok := vars[name]
		return v, ok
	}
}

func TestExpandVars(t *testing.T) {
	t.Parallel()

	lookup := fakeLookup(map[string]string{"HOST": "db.local", "EMPTY": ""})

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "no reference", in: "plain", want: "plain"},
		{name: "set variable", in: "postgres://${HOST}/app", want: "postgres://db.local/app"},
		{name: "unset variable kept literal", in: "${MISSING}", want: "${MISSING}"},
		{name: "default used when unset", in: "${MISSING:-fallback}", want: "fallback"},
		{name: "empty default", in: "x${MISSING:-}y", want: "xy"},
		{name: "set variable wins over default", in: "${HOST:-other}", want: "db.local"},
		{name: "set but empty variable wins over default", in: "${EMPTY:-other}", want: ""},
		{name: "several references", in: "${HOST}:${PORT:-5432}", want: "db.local:5432"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := expandVars(tt.in, lookup); got != tt.want {
				t.Errorf("expandVars(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestExpandConfig is the F275 regression: Claude Code expands variables in
// command, args, url and headers, not only env, so the probe must too.
func TestExpandConfig(t *testing.T) {
	t.Parallel()

	lookup := fakeLookup(map[string]string{"BIN": "server", "DB": "postgres://x", "TOKEN": "s3cret"})
	in := ServerConfig{
		Command: "${BIN}",
		Args:    []string{"--db", "${DB}", "--mode=${MODE:-ro}"},
		URL:     "https://${HOST:-example.com}/mcp",
		Headers: map[string]string{"Authorization": "Bearer ${TOKEN}"},
	}
	origArgs := slices.Clone(in.Args)

	got := expandConfig(in, lookup)

	if got.Command != "server" {
		t.Errorf("Command = %q", got.Command)
	}
	if want := []string{"--db", "postgres://x", "--mode=ro"}; !slices.Equal(got.Args, want) {
		t.Errorf("Args = %q, want %q", got.Args, want)
	}
	if got.URL != "https://example.com/mcp" {
		t.Errorf("URL = %q", got.URL)
	}
	if got.Headers["Authorization"] != "Bearer s3cret" {
		t.Errorf("Authorization header = %q", got.Headers["Authorization"])
	}
	if !slices.Equal(in.Args, origArgs) || in.Headers["Authorization"] != "Bearer ${TOKEN}" {
		t.Error("expandConfig mutated its input")
	}
}

func TestBuildProcessEnv_ExpandsDefaults(t *testing.T) {
	t.Setenv("QSDEV_TEST_EXPAND_SET", "value")

	env := buildProcessEnv(map[string]string{
		"A": "${QSDEV_TEST_EXPAND_SET}",
		"B": "${QSDEV_TEST_EXPAND_UNSET_987:-fallback}",
	})

	for _, want := range []string{"A=value", "B=fallback"} {
		if !slices.Contains(env, want) {
			t.Errorf("process env missing %q", want)
		}
	}
}

// TestCheckServer_HTTPSendsHeaders verifies configured headers (with variables
// expanded) reach an HTTP server, so servers that need an Authorization header
// are not reported unreachable.
func TestCheckServer_HTTPSendsHeaders(t *testing.T) {
	t.Setenv("QSDEV_TEST_MCP_TOKEN", "tok")

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mcpReply(r))
	}
	srv := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h := CheckServer(ctx, ServerConfig{
		Name:    "auth",
		URL:     srv.URL,
		Headers: map[string]string{"Authorization": "Bearer ${QSDEV_TEST_MCP_TOKEN}"},
	})
	if h.Status != StatusHealthy {
		t.Errorf("status = %q, want healthy (error=%q)", h.Status, h.Error)
	}
}
