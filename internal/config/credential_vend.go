package config

import (
	"regexp"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// awsRoleARNPattern matches an IAM role ARN in any AWS partition, including a
// role path. It admits no wildcard: security.credential_vend.aws.role_arns is
// matched exactly.
var awsRoleARNPattern = regexp.MustCompile(`^arn:aws(?:-[a-z]+)*:iam::[0-9]{12}:role/[A-Za-z0-9+=,.@_/-]+$`)

// gcpServiceAccountPattern accepts the identifiers the IAM Credentials API takes
// for a service account: its email address, or its numeric unique ID. It rejects
// anything carrying URL syntax ("/", "?", "#", ":"), which would otherwise let
// the value rewrite the ADC-authorized request target (e.g. append ":signJwt#"
// to call a different IAM method).
var gcpServiceAccountPattern = regexp.MustCompile(`^(?:[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+|[0-9]+)$`)

// azureClientIDPattern matches a managed identity's client ID (a GUID).
var azureClientIDPattern = regexp.MustCompile(`^[0-9A-Fa-f]{8}(?:-[0-9A-Fa-f]{4}){3}-[0-9A-Fa-f]{12}$`)

// azureScopePattern matches a token scope: a non-empty value with no
// whitespace or control characters.
var azureScopePattern = regexp.MustCompile(`^[^\s\x00-\x1f\x7f]+$`)

// ValidGCPServiceAccount reports whether s is a GCP service-account email or
// numeric unique ID, the only forms qsdev_credential_vend impersonates.
func ValidGCPServiceAccount(s string) bool {
	return gcpServiceAccountPattern.MatchString(s)
}

// validateCredentialVend checks the security.credential_vend allow-lists. Each
// entry is compared exactly against a tool request, so a malformed one (a
// wildcard, a typo) can never match and would silently deny the identity the
// operator meant to allow.
func validateCredentialVend(cv types.CredentialVendConfig) []ValidationError {
	const field = "security.credential_vend."
	var errs []ValidationError
	check := func(name string, values []string, re *regexp.Regexp, msg string) {
		for _, v := range values {
			if !re.MatchString(v) {
				errs = append(errs, ValidationError{Field: field + name, Value: v, Message: msg})
			}
		}
	}
	check("aws.role_arns", cv.AWS.RoleARNs, awsRoleARNPattern,
		"not an IAM role ARN (arn:aws:iam::<account-id>:role/<name>); entries match exactly, wildcards are not supported")
	check("gcp.service_accounts", cv.GCP.ServiceAccounts, gcpServiceAccountPattern,
		"not a service-account email or numeric unique ID")
	check("azure.scopes", cv.Azure.Scopes, azureScopePattern,
		"not a token scope (e.g. https://management.azure.com/.default)")
	check("azure.identities", cv.Azure.Identities, azureClientIDPattern,
		"not a managed-identity client ID (a GUID)")
	return errs
}
