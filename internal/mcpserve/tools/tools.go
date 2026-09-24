// Package tools aggregates the security and development-environment MCP tool
// registrations (Phase 32, Unit 32.9) into a single set the universal server
// mounts. The seven tools, two of them opt-in (see Options), span four
// sub-packages — security/ (credential_vend, security_scan, policy_check),
// devenv/ (env_info, nix_run), and status/ (qsdev_status,
// qsdev_devenv_doctor) — with shared helpers in toolutil/.
//
// Every registration is fully implemented and delegates to real qsdev packages
// or external APIs; none is a stub. Each is tagged with the middleware Category
// that governs its rate limit and content-safety handling.
package tools

import (
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/security"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/status"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Options selects the opt-in tools All registers. The zero value registers
// neither, so a caller that forgets to set them fails closed.
type Options struct {
	// CredentialVend is the project's security.credential_vend.
	// qsdev_credential_vend is registered only when it is enabled, and vends
	// only what its allow-lists name.
	CredentialVend types.CredentialVendConfig
	// NixRun registers qsdev_nix_run.
	NixRun bool
}

// All returns the security and devenv tool registrations bound to projectRoot
// that opts selects, ready to be mounted on the universal server. The order is
// stable: security, then devenv, then status. enforced is the Guardrail policy
// installed in the server's middleware chain (nil when nothing is narrowed);
// qsdev_policy_check reports from it so "reported enforced" is exactly what is
// enforced.
func All(projectRoot string, enforced *middleware.Policy, opts Options) []spi.ToolRegistration {
	var regs []spi.ToolRegistration
	regs = append(regs, security.Tools(projectRoot, enforced, opts.CredentialVend)...)
	regs = append(regs, devenv.Tools(projectRoot, opts.NixRun)...)
	regs = append(regs, status.Tools(projectRoot)...)
	return regs
}

// Names returns the name of every tool All can register, whatever the
// Options, in All's order.
func Names() []string {
	every := Options{CredentialVend: types.CredentialVendConfig{Enabled: true}, NixRun: true}
	regs := All("", nil, every)
	names := make([]string, len(regs))
	for i, r := range regs {
		names[i] = r.Name
	}
	return names
}
