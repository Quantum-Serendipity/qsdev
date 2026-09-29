package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestCloudConfig_IsolateCLIConfig checks cloud.isolate_cli_config (W135)
// parses, survives the config -> answers -> config round trip join and init
// use, and has a single source, the committed .qsdev.yaml: Apply installs it
// over the saved answers, re-creating a project and a day-2 sync keep it, any
// organization or project layer can turn it on, and .qsdev.local.yaml cannot
// set it.
func TestCloudConfig_IsolateCLIConfig(t *testing.T) {
	t.Parallel()
	const committedYAML = "version: 2\ntier: standard\ncloud:\n  isolate_cli_config: true\n"
	committed, err := ParseQsdevConfigBytes([]byte(committedYAML))
	if err != nil {
		t.Fatal(err)
	}
	if !committed.Cloud.IsolateCLIConfig {
		t.Fatal("cloud.isolate_cli_config did not parse")
	}

	t.Run("round trip", func(t *testing.T) {
		t.Parallel()
		answers := ConfigToAnswers(committed, types.DetectedProject{}, "/proj")
		if !answers.Cloud.IsolateCLIConfig {
			t.Fatal("ConfigToAnswers dropped cloud.isolate_cli_config")
		}
		back := AnswersToConfig(answers, "")
		if !back.Cloud.IsolateCLIConfig {
			t.Fatal("AnswersToConfig dropped cloud.isolate_cli_config")
		}
		data, err := MarshalProjectConfig(back)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "cloud:\n    isolate_cli_config: true\n") {
			t.Errorf("marshaled config lacks the cloud block:\n%s", data)
		}
		off, err := MarshalProjectConfig(types.QsdevConfig{Version: types.ConfigVersionCurrent})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(off), "cloud") {
			t.Errorf("the default (off) writes a cloud key:\n%s", off)
		}
	})

	t.Run("apply", func(t *testing.T) {
		t.Parallel()
		for _, tt := range []struct {
			name      string
			committed *types.QsdevConfig
			want      bool
		}{
			{"committed on", committed, true},
			{"committed off", &types.QsdevConfig{Version: types.ConfigVersionCurrent}, false},
		} {
			policy, err := ResolveProjectPolicy(tt.committed, nil)
			if err != nil {
				t.Fatal(err)
			}
			a := types.WizardAnswers{Cloud: types.CloudConfig{IsolateCLIConfig: !tt.want}}
			policy.Apply(&a)
			if a.Cloud.IsolateCLIConfig != tt.want {
				t.Errorf("%s: applied isolate_cli_config = %v, want %v", tt.name, a.Cloud.IsolateCLIConfig, tt.want)
			}
		}
	})

	t.Run("preserve on re-create", func(t *testing.T) {
		t.Parallel()
		fresh := types.QsdevConfig{}
		PreserveCommittedPolicy(&fresh, committed)
		if !fresh.Cloud.IsolateCLIConfig {
			t.Error("re-creating the project dropped cloud.isolate_cli_config")
		}
	})

	t.Run("sync keeps it", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, ".qsdev.yaml")
		if err := os.WriteFile(path, []byte(committedYAML), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := SyncProjectConfig(dir, types.WizardAnswers{Tier: "full"}); err != nil {
			t.Fatal(err)
		}
		synced, err := ParseQsdevConfig(path)
		if err != nil {
			t.Fatal(err)
		}
		if synced.Tier != "full" {
			t.Fatalf("sync did not run: tier = %q", synced.Tier)
		}
		if !synced.Cloud.IsolateCLIConfig {
			t.Error("sync dropped cloud.isolate_cli_config")
		}
	})

	t.Run("layers only turn it on", func(t *testing.T) {
		t.Parallel()
		on := &types.QsdevConfig{Cloud: types.CloudConfig{IsolateCLIConfig: true}}
		off := &types.QsdevConfig{}
		for _, tt := range []struct {
			name         string
			org, project *types.QsdevConfig
			want         bool
		}{
			{"org on", on, off, true},
			{"project on", off, on, true},
			{"both off", off, off, false},
		} {
			resolved, err := ResolveConfig(tt.org, tt.project, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := resolved.Config.Cloud.IsolateCLIConfig; got != tt.want {
				t.Errorf("%s: resolved isolate_cli_config = %v, want %v", tt.name, got, tt.want)
			}
		}
	})

	t.Run("local key rejected", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), ".qsdev.local.yaml")
		if err := os.WriteFile(path, []byte("cloud:\n  isolate_cli_config: true\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := ParseLocalConfig(path); err == nil {
			t.Error("a cloud key in .qsdev.local.yaml parsed; devenv.nix is committed, so the setting is team-wide")
		}
	})
}
