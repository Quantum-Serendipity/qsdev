package mcpserve

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools"
)

// TestMountTools proves the seven security and devenv tools (Unit 32.9) are
// registered on the underlying mcp-go server and therefore appear in tools/list.
func TestMountTools(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	srv := New(WithProjectRoot(dir))
	srv.MountTools(tools.All(dir))

	listed := srv.MCPServer().ListTools()
	want := []string{
		"qsdev_credential_vend",
		"qsdev_security_scan",
		"qsdev_policy_check",
		"qsdev_env_info",
		"qsdev_nix_run",
		"qsdev_status",
		"qsdev_devenv_doctor",
	}
	for _, name := range want {
		if _, ok := listed[name]; !ok {
			t.Errorf("tool %q not mounted; mounted=%v", name, keys(listed))
		}
	}
}
