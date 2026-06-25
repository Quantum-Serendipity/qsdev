// Package tools aggregates the security and development-environment MCP tool
// registrations (Phase 32, Unit 32.9) into a single set the universal server
// mounts. The seven tools span four sub-packages — security/ (credential_vend,
// security_scan, policy_check), devenv/ (env_info, nix_run), and status/
// (qsdev_status, qsdev_devenv_doctor) — with shared helpers in toolutil/.
//
// Every registration is fully implemented and delegates to real qsdev packages
// or external APIs; none is a stub. Each is tagged with the middleware Category
// that governs its rate limit and content-safety handling.
package tools

import (
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/security"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/status"
)

// All returns every security and devenv tool registration bound to projectRoot,
// ready to be mounted on the universal server. The order is stable: security,
// then devenv, then status.
func All(projectRoot string) []spi.ToolRegistration {
	var regs []spi.ToolRegistration
	regs = append(regs, security.Tools(projectRoot)...)
	regs = append(regs, devenv.Tools(projectRoot)...)
	regs = append(regs, status.Tools(projectRoot)...)
	return regs
}
