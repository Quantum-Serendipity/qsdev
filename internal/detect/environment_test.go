package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectEnvironment_DevenvNix(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "devenv.nix"), "{ pkgs, ... }: {}")

	env := detectEnvironment(dir)
	if !env.HasDevenvNix {
		t.Error("expected HasDevenvNix to be true")
	}
}

func TestDetectEnvironment_DevenvYaml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "devenv.yaml"), "inputs:\n  nixpkgs:\n    url: github:NixOS/nixpkgs/nixpkgs-unstable")

	env := detectEnvironment(dir)
	if !env.HasDevenvYaml {
		t.Error("expected HasDevenvYaml to be true")
	}
}

func TestDetectEnvironment_ClaudeDir(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.Mkdir(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(claudeDir, "settings.json"), `{"permissions":{}}`)

	env := detectEnvironment(dir)
	if !env.HasClaudeDir {
		t.Error("expected HasClaudeDir to be true")
	}
	if !env.HasClaudeSettings {
		t.Error("expected HasClaudeSettings to be true")
	}
}

func TestDetectEnvironment_ClaudeMd(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), "# CLAUDE.md")

	env := detectEnvironment(dir)
	if !env.HasClaudeMd {
		t.Error("expected HasClaudeMd to be true")
	}
}

func TestDetectEnvironment_Envrc(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".envrc"), "use devenv")

	env := detectEnvironment(dir)
	if !env.HasEnvrc {
		t.Error("expected HasEnvrc to be true")
	}
}

func TestDetectEnvironment_McpJson(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".mcp.json"), "{}")

	env := detectEnvironment(dir)
	if !env.HasMcpJson {
		t.Error("expected HasMcpJson to be true")
	}
}

func TestDetectEnvironment_GitRepo(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(gitDir, "config"),
		`[core]
	repositoryformatversion = 0
	filemode = true
[remote "origin"]
	url = git@github.com:example/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
[branch "main"]
	remote = origin
	merge = refs/heads/main
`)

	env := detectEnvironment(dir)
	if !env.IsGitRepo {
		t.Error("expected IsGitRepo to be true")
	}
	if env.RemoteURL != "git@github.com:example/repo.git" {
		t.Errorf("RemoteURL = %q, want %q", env.RemoteURL, "git@github.com:example/repo.git")
	}
}

func TestDetectEnvironment_GitHooks(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write an executable hook (not a .sample file).
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	env := detectEnvironment(dir)
	if !env.HasGitHooks {
		t.Error("expected HasGitHooks to be true")
	}
}

func TestDetectEnvironment_GitHooksSampleOnly(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Only .sample files should not count.
	samplePath := filepath.Join(hooksDir, "pre-commit.sample")
	if err := os.WriteFile(samplePath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	env := detectEnvironment(dir)
	if env.HasGitHooks {
		t.Error("expected HasGitHooks to be false when only .sample hooks exist")
	}
}

func TestDetectEnvironment_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	env := detectEnvironment(dir)
	if env.HasDevenvNix || env.HasDevenvYaml || env.HasClaudeDir || env.HasClaudeMd ||
		env.HasClaudeSettings || env.HasEnvrc || env.HasMcpJson || env.IsGitRepo ||
		env.HasGitHooks || env.RemoteURL != "" {
		t.Errorf("empty dir should have all-false state, got %+v", env)
	}
}

func TestDetectEnvironment_HTTPSRemote(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(gitDir, "config"),
		`[remote "origin"]
	url = https://github.com/example/repo.git
`)

	env := detectEnvironment(dir)
	if env.RemoteURL != "https://github.com/example/repo.git" {
		t.Errorf("RemoteURL = %q, want HTTPS URL", env.RemoteURL)
	}
}

// writeFile is a test helper that creates a file with the given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
