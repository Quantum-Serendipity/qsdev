package mcpregistry

// CriterionResult records whether a single compliance criterion was met.
type CriterionResult struct {
	Name   string
	Passed bool
	Detail string
}

// GradeResult holds the computed compliance level and the individual criterion
// outcomes that produced it.
type GradeResult struct {
	Level    ComplianceLevel
	Criteria []CriterionResult
}

// GradeServer evaluates a server definition against the compliance ladder and
// returns the highest fully-satisfied level along with per-criterion details.
// Every criterion except external-attestation is pure; the result is
// deterministic given the injected AttestationChecker (which the claudecode
// addon wires to a contentsign-backed verifier at startup, and which defaults
// to a no-op returning false).
func GradeServer(def *McpServerDefinition) GradeResult {
	var criteria []CriterionResult

	// Basic — always satisfied.
	level := ComplianceBasic

	// Standard criteria.
	noSecrets := !hasPlaintextSecrets(def)
	criteria = append(criteria, CriterionResult{
		Name:   "no-plaintext-secrets",
		Passed: noSecrets,
		Detail: boolDetail(noSecrets, "env values contain no plaintext secrets", "env values may contain plaintext secrets"),
	})

	stdioTransport := def.Transport == TransportStdio
	criteria = append(criteria, CriterionResult{
		Name:   "stdio-transport",
		Passed: stdioTransport,
		Detail: boolDetail(stdioTransport, "transport is stdio", "transport is "+string(def.Transport)),
	})

	standardMet := noSecrets && stdioTransport
	if standardMet {
		level = ComplianceStandard
	}

	// Secure criteria.
	localOnly := isLocalOnly(def)
	criteria = append(criteria, CriterionResult{
		Name:   "local-only",
		Passed: localOnly,
		Detail: boolDetail(localOnly, "command runs locally", "command may fetch from network"),
	})

	noNpxY := !hasNpxDashY(def)
	criteria = append(criteria, CriterionResult{
		Name:   "no-npx-dash-y",
		Passed: noNpxY,
		Detail: boolDetail(noNpxY, "no npx -y auto-install", "npx -y enables auto-install of unreviewed packages"),
	})

	secureMet := standardMet && localOnly && noNpxY
	if secureMet {
		level = ComplianceSecure
	}

	// Verified criteria.
	provenance := hasVerifiedProvenance(def)
	criteria = append(criteria, CriterionResult{
		Name:   "verified-provenance",
		Passed: provenance,
		Detail: boolDetail(provenance, "command has verified provenance", "command provenance is unverified"),
	})

	verifiedMet := secureMet && provenance
	if verifiedMet {
		level = ComplianceVerified
	}

	// Attested criteria. Attestation only ever lifts a server that already
	// reached Verified, which requires hasVerifiedProvenance (a /nix/store path
	// or the qsdev binary). External npx/uvx doc servers fail the earlier
	// local-only and provenance criteria, so they can never reach Attested even
	// with a valid signature. Gate the (expensive, binary-streaming) check on
	// verifiedMet so it is skipped for servers that cannot reach Attested — but
	// keep the per-criterion report honest by distinguishing "not evaluated"
	// from "evaluated, no signature": folding the gate into Passed would tell an
	// operator a validly-signed sub-Verified server has "no signature".
	attestationPassed := false
	var attestationDetail string
	switch {
	case !verifiedMet:
		attestationDetail = "not evaluated (server has not reached Verified)"
	case hasExternalAttestation(def):
		attestationPassed = true
		attestationDetail = "verified attestation signature present"
	default:
		attestationDetail = "no verified attestation signature"
	}
	criteria = append(criteria, CriterionResult{
		Name:   "external-attestation",
		Passed: attestationPassed,
		Detail: attestationDetail,
	})

	// attestationPassed already implies verifiedMet (the !verifiedMet branch
	// leaves it false), so it is the full Attested gate.
	if attestationPassed {
		level = ComplianceAttested
	}

	return GradeResult{
		Level:    level,
		Criteria: criteria,
	}
}

// boolDetail returns trueMsg when cond is true, falseMsg otherwise.
func boolDetail(cond bool, trueMsg, falseMsg string) string {
	if cond {
		return trueMsg
	}
	return falseMsg
}
