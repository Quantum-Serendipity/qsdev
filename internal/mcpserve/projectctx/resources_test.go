package projectctx

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// readResource finds a resource by URI and invokes its handler.
func readResource(t *testing.T, pc *ProjectContext, uri string) *spi.ResourceResult {
	t.Helper()
	for _, reg := range pc.Resources() {
		if reg.URI == uri {
			res, err := reg.Handler(context.Background(), &spi.ToolCallContext{}, &spi.ResourceRequest{URI: uri})
			if err != nil {
				t.Fatalf("resource %s handler error: %v", uri, err)
			}
			if len(res.Contents) == 0 {
				t.Fatalf("resource %s returned no contents", uri)
			}
			return res
		}
	}
	t.Fatalf("resource %s not registered", uri)
	return nil
}

func TestResourceMIMETypes(t *testing.T) {
	t.Parallel()
	dir, pc := newGoProject(t)
	writeFile(t, dir, ".devinit/.qsdev-init-state.yaml", "files: {}\n")

	cases := []struct {
		uri      string
		wantMIME string
		wantJSON bool
	}{
		{resURIDetection, mimeJSON, true},
		{resURIConfig, mimeYAML, false},
		{resURIState, mimeYAML, false},
		{resURIMCPServers, mimeJSON, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.uri, func(t *testing.T) {
			t.Parallel()
			c := readResource(t, pc, tc.uri).Contents[0]
			if c.MIMEType != tc.wantMIME {
				t.Errorf("MIME = %q, want %q", c.MIMEType, tc.wantMIME)
			}
			if tc.wantJSON {
				var v any
				if err := json.Unmarshal([]byte(c.Text), &v); err != nil {
					t.Errorf("content is not valid JSON: %v", err)
				}
			}
		})
	}
}

func TestResourceConfigContent(t *testing.T) {
	t.Parallel()
	_, pc := newGoProject(t)
	c := readResource(t, pc, resURIConfig).Contents[0]
	if c.Text != validQsdevYAML {
		t.Errorf("config resource text = %q, want raw file %q", c.Text, validQsdevYAML)
	}
}

func TestResourceConfigMissingDegrades(t *testing.T) {
	t.Parallel()
	pc, err := NewProjectContext(t.TempDir())
	if err != nil {
		t.Fatalf("NewProjectContext: %v", err)
	}
	c := readResource(t, pc, resURIConfig).Contents[0]
	if c.MIMEType != mimeYAML {
		t.Errorf("MIME = %q, want %q even when missing", c.MIMEType, mimeYAML)
	}
	if !strings.Contains(c.Text, "not_configured") {
		t.Errorf("missing config should degrade to a not_configured comment, got %q", c.Text)
	}
}

func TestPackageContextNotConfigured(t *testing.T) {
	t.Parallel()
	_, pc := newGoProject(t)
	// The per-package resource is registered under a templated URI; invoke its
	// handler directly with a concrete package URI.
	var handler spi.ResourceHandler
	for _, reg := range pc.Resources() {
		if reg.URI == resURIPackageCtx {
			handler = reg.Handler
		}
	}
	if handler == nil {
		t.Fatal("per-package context resource not registered")
	}
	res, err := handler(context.Background(), &spi.ToolCallContext{},
		&spi.ResourceRequest{URI: "qsdev://project/go:example/context"})
	if err != nil {
		t.Fatalf("package context handler error: %v", err)
	}
	c := res.Contents[0]
	if c.MIMEType != mimeJSON {
		t.Errorf("MIME = %q, want %q", c.MIMEType, mimeJSON)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(c.Text), &payload); err != nil {
		t.Fatalf("package context not valid JSON: %v", err)
	}
	if payload["status"] != "not_configured" {
		t.Errorf("status = %v, want not_configured", payload["status"])
	}
}
