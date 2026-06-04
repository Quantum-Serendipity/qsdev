package contracttest

import (
	"context"
	"runtime"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

func TestHookDeployer(t *testing.T, deployer aiframework.HookDeployer, fixtures ContractFixtures) {
	t.Helper()

	t.Run("SupportedEventsNonEmpty", func(t *testing.T) {
		events := deployer.SupportedEvents()
		if len(events) == 0 {
			t.Error("SupportedEvents() returned empty")
		}
	})

	t.Run("ProtocolNonZero", func(t *testing.T) {
		_ = deployer.Protocol()
	})

	t.Run("DeployProducesFiles", func(t *testing.T) {
		policies := []aiframework.HookPolicy{
			{
				Event:        aiframework.EventPreToolUse,
				ToolMatchers: []string{"Bash"},
				Logic:        aiframework.LogicPackageGuard,
				Timeout:      30,
			},
		}
		files, err := deployer.Deploy(context.Background(), policies)
		if err != nil {
			t.Fatalf("Deploy() error: %v", err)
		}
		if len(files) == 0 {
			t.Error("Deploy() produced no files")
		}
		if runtime.GOOS != "windows" {
			for _, f := range files {
				if f.Mode&0o111 == 0 {
					t.Errorf("hook file %q is not executable (mode %o)", f.Path, f.Mode)
				}
			}
		}
	})

	t.Run("UndeployNoError", func(t *testing.T) {
		dir := t.TempDir()
		if err := deployer.Undeploy(context.Background(), dir); err != nil {
			t.Errorf("Undeploy() error: %v", err)
		}
	})
}
