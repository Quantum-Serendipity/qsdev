package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// Resources returns the two Claude Code resources: the active settings.json and
// the deployed hook listing (with content). Both source real on-disk or rendered
// data and degrade to a structured not_configured JSON document when their
// prerequisites are absent.
func (a *Adapter) Resources() []spi.ResourceRegistration {
	return []spi.ResourceRegistration{
		{
			URI:         resURISettings,
			Name:        "Claude Code settings",
			Description: "The active Claude Code settings.json: the deployed .claude/settings.json when present, otherwise the settings rendered from the project's qsdev policy.",
			MIMEType:    mimeJSON,
			Handler:     a.readSettings,
		},
		{
			URI:         resURIHooks,
			Name:        "Claude Code hooks",
			Description: "The Claude Code hook scripts deployed under .claude/hooks/, each with its filesystem metadata, settings.json wiring, and full script content.",
			MIMEType:    mimeJSON,
			Handler:     a.readHooks,
		},
	}
}

// readSettings returns the project's active settings.json. It prefers the
// deployed .claude/settings.json (authoritative for what is actually active) and
// falls back to rendering the canonical settings from policy. A render failure
// with no deployed file degrades to a not_configured document.
func (a *Adapter) readSettings(ctx context.Context, cc *spi.ToolCallContext, _ *spi.ResourceRequest) (*spi.ResourceResult, error) {
	deployedPath := filepath.Join(cc.ProjectRoot, ".claude", "settings.json")
	if data, err := os.ReadFile(deployedPath); err == nil {
		return jsonResource(resURISettings, data), nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading %s: %w", deployedPath, err)
	}

	files, err := a.ref.Render(ctx, a.policyInputFor(cc.ProjectRoot))
	if err != nil {
		return notConfiguredResource(resURISettings,
			"no deployed .claude/settings.json and rendering from policy failed",
			map[string]any{"error": err.Error()})
	}
	for _, f := range files {
		if filepath.Base(f.Path) == "settings.json" {
			return jsonResource(resURISettings, f.Content), nil
		}
	}
	return notConfiguredResource(resURISettings,
		"no settings.json deployed or rendered for this project", nil)
}

// readHooks lists every deployed hook script with its metadata, settings.json
// wiring, and content. A missing hooks directory degrades to not_configured.
func (a *Adapter) readHooks(_ context.Context, cc *spi.ToolCallContext, _ *spi.ResourceRequest) (*spi.ResourceResult, error) {
	hooksDir := filepath.Join(cc.ProjectRoot, ".claude", "hooks")
	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return notConfiguredResource(resURIHooks, "no claude code hooks deployed",
				map[string]any{"hooks_dir": hooksDir})
		}
		return nil, fmt.Errorf("reading hooks directory %s: %w", hooksDir, err)
	}

	wiring := deployedHookWiring(cc.ProjectRoot)
	hooks := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		entry := hookFileEntry(hooksDir, e, wiring[e.Name()])
		if content, rerr := os.ReadFile(filepath.Join(hooksDir, e.Name())); rerr == nil {
			entry["content"] = string(content)
		}
		hooks = append(hooks, entry)
	}
	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i]["name"].(string) < hooks[j]["name"].(string)
	})

	payload := map[string]any{"hooks_dir": hooksDir, "count": len(hooks), "hooks": hooks}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling hooks listing: %w", err)
	}
	return jsonResource(resURIHooks, data), nil
}

// jsonResource wraps JSON bytes in a single-content ResourceResult.
func jsonResource(uri string, data []byte) *spi.ResourceResult {
	return &spi.ResourceResult{Contents: []spi.ResourceContent{{
		URI: uri, MIMEType: mimeJSON, Text: string(data),
	}}}
}

// notConfiguredResource returns a structured not_configured JSON document for a
// resource read whose prerequisite is absent (resources carry no IsError flag,
// so the payload itself signals the degraded state).
func notConfiguredResource(uri, reason string, extra map[string]any) (*spi.ResourceResult, error) {
	payload := map[string]any{"status": "not_configured", "reason": reason, "uri": uri}
	for k, v := range extra {
		payload[k] = v
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling not_configured payload: %w", err)
	}
	return jsonResource(uri, data), nil
}
