// Package security implements the security-oriented MCP tool handlers for the
// universal qsdev server (Phase 32, Unit 32.9): credential_vend (short-lived
// cloud credential vending via AWS STS, GCP IAM Credentials, and Azure Managed
// Identity), security_scan (OSV.dev vulnerability scanning of lock-file
// dependencies), and policy_check (in-memory security-policy evaluation).
//
// Every handler degrades gracefully: a missing provider, lock file, or policy
// file yields a structured not_configured result rather than an error or a
// crash. credential_vend is tagged CategoryCredential (for rate limiting) and,
// as the sole tool registered under middleware.CredentialVendToolName, is the
// only surface exempt from ContentSafety redaction so its short-lived token
// output survives — the exemption is keyed on that trusted tool identity, not on
// the self-declared category, so no other tool can borrow the exemption.
package security

import (
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// Tier values mirror projectctx's pruning tiers: lower is more core. policy_check
// is a fast-path critical tool; the external-API tools are standard tier.
const (
	tierCritical = 0
	tierStandard = 1
)

// Tools returns the three security tool registrations bound to projectRoot.
func Tools(projectRoot string) []spi.ToolRegistration {
	cv := newCredentialVendor()
	scanner := newSecurityScanner(projectRoot)
	checker := newPolicyChecker(projectRoot)

	return []spi.ToolRegistration{
		{
			Name:        middleware.CredentialVendToolName,
			Description: "Vend short-lived cloud credentials by exchanging the host's ambient identity: AWS STS (AssumeRole/GetSessionToken), GCP IAM Credentials (service-account access token), or Azure Managed Identity. Returns only time-boxed credential material, never long-lived secrets.",
			InputSchema: credentialVendSchema(),
			Category:    middleware.CategoryCredential,
			Tier:        tierStandard,
			Handler:     cv.handle,
		},
		{
			Name:        "qsdev_security_scan",
			Description: "Scan the project's pinned dependencies (from go.sum, package-lock.json, Cargo.lock, poetry.lock, uv.lock, or requirements.txt) against the OSV.dev vulnerability database and report findings at or above a severity threshold.",
			InputSchema: securityScanSchema(),
			Category:    middleware.CategorySecurity,
			Tier:        tierStandard,
			Handler:     scanner.handle,
		},
		{
			Name:        "qsdev_policy_check",
			Description: "Evaluate the project's security policy entirely in memory from .qsdev.yaml (with the .qsdev.local.yaml overlay). Returns allowed/denied/ask for a named tool plus the applicable rules and each rule's source (project, local, or default).",
			InputSchema: policyCheckSchema(),
			Category:    middleware.CategoryPolicy,
			Tier:        tierCritical,
			Handler:     checker.handle,
		},
	}
}

func credentialVendSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"provider": map[string]any{
				"type":        "string",
				"enum":        []any{"aws", "gcp", "azure"},
				"description": "Cloud provider to vend credentials from.",
			},
			"role_arn":        map[string]any{"type": "string", "description": "AWS only: role ARN to AssumeRole into. Omit to use GetSessionToken."},
			"service_account": map[string]any{"type": "string", "description": "GCP only: service-account email to impersonate."},
			"identity":        map[string]any{"type": "string", "description": "Azure only: user-assigned managed-identity client id. Omit for the system-assigned identity."},
			"scope":           map[string]any{"type": "string", "description": "Azure only: token scope/audience (default https://management.azure.com/.default)."},
			"ttl":             map[string]any{"type": "string", "description": "Requested credential lifetime as a Go duration (e.g. \"1h\") or seconds. Default 1h; AWS is clamped to 15m..12h, GCP to <=1h."},
		},
		"required": []any{"provider"},
	}
}

func securityScanSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"manifest_path": map[string]any{"type": "string", "description": "Explicit lock-file path. Omit to auto-detect a lock file under the project root."},
			"severity_threshold": map[string]any{
				"type":        "string",
				"enum":        []any{"low", "medium", "high", "critical"},
				"description": "Minimum severity to report (default medium). Vulnerabilities of unknown severity are always reported.",
			},
		},
	}
}

func policyCheckSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"policy_path": map[string]any{"type": "string", "description": "Path to the policy file (default the project's .qsdev.yaml)."},
			"tool_name":   map[string]any{"type": "string", "description": "Evaluate the verdict for this specific tool. Omit to list all rules."},
		},
	}
}
