package claudecode

import (
	"encoding/json"
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/lsp"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// lspPluginManifest is the .claude-plugin/plugin.json manifest that makes the
// qsdev-lsp skill directory auto-load as a Claude Code plugin.
type lspPluginManifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// lspJSONEntry is a single language entry in .lsp.json, keyed by LSP language
// id at the top level. It deliberately omits a top-level "name" key and any
// "filePatterns" key: Claude Code derives language association from
// extensionToLanguage, which is required.
type lspJSONEntry struct {
	Command               string            `json:"command"`
	Args                  []string          `json:"args,omitempty"`
	ExtensionToLanguage   map[string]string `json:"extensionToLanguage"`
	InitializationOptions map[string]any    `json:"initializationOptions,omitempty"`
	Settings              map[string]any    `json:"settings,omitempty"`
}

// lspPluginManifestPath is the fixed manifest path for the consolidated plugin.
const lspPluginManifestPath = ".claude/skills/qsdev-lsp/.claude-plugin/plugin.json"

// lspJSONPath is the fixed path for the LSP server configuration object.
const lspJSONPath = ".claude/skills/qsdev-lsp/.lsp.json"

// GenerateLspPlugin produces the consolidated Claude Code LSP plugin: a manifest
// and a .lsp.json keyed by LSP language id. It auto-includes every default-on
// server whose ecosystem was detected, plus the always-on nixd server. Opt-in
// (default-off) servers such as kotlin are never auto-included, even when their
// ecosystem is detected.
func GenerateLspPlugin(answers types.WizardAnswers, reg *lsp.LSPRegistry) ([]types.GeneratedFile, error) {
	entries := make(map[string]lspJSONEntry)

	// Collect default-on servers for every detected ecosystem.
	for _, lang := range answers.Languages {
		cfg, ok := reg.ByEcosystem(lang.Name)
		if !ok || !cfg.DefaultOn {
			continue
		}
		entries[cfg.LanguageID] = lspConfigToJSONEntry(cfg)
	}

	// nixd is always-on for every qsdev project, regardless of detected
	// languages.
	if nix, ok := reg.ByEcosystem("nix"); ok {
		entries[nix.LanguageID] = lspConfigToJSONEntry(nix)
	}

	// encoding/json sorts map keys, so the marshaled output is deterministic.
	lspBytes, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling .lsp.json: %w", err)
	}
	lspBytes = append(lspBytes, '\n')

	manifest := lspPluginManifest{
		Name:        "qsdev-lsp",
		Description: "qsdev-managed LSP servers for configured ecosystems",
		Version:     "0.1.0",
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling plugin.json: %w", err)
	}
	manifestBytes = append(manifestBytes, '\n')

	return []types.GeneratedFile{
		{
			Path:     lspPluginManifestPath,
			Content:  manifestBytes,
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Overwrite,
			Owner:    "lsp-config",
		},
		{
			Path:     lspJSONPath,
			Content:  lspBytes,
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Overwrite,
			Owner:    "lsp-config",
		},
	}, nil
}

// lspConfigToJSONEntry builds a single .lsp.json language entry from a registry
// config. Empty Args/InitOptions/Settings are omitted by the struct's omitempty
// tags; extensionToLanguage is always populated.
func lspConfigToJSONEntry(cfg *lsp.LSPServerConfig) lspJSONEntry {
	return lspJSONEntry{
		Command:               cfg.Command,
		Args:                  cfg.Args,
		ExtensionToLanguage:   cfg.ExtensionToLanguage(),
		InitializationOptions: cfg.InitOptions,
		Settings:              cfg.Settings,
	}
}
