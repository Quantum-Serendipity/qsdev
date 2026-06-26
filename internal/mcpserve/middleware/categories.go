package middleware

// Tool category taxonomy.
//
// The set below is intentionally limited to the categories that are actually
// assigned to a registered tool, depended on by the Guardrail default policy, or
// exercised by the rate-limit/guardrail tests — plus CategoryGeneral, the
// default bucket for uncategorized or unknown categories. An earlier draft
// carried a speculative 19-category taxonomy (the design spike that would have
// pinned the full list, research-spikes/gdev-universal-mcp-server-design, is not
// in the repository); the unassigned categories and their dead rate-limit table
// entries were pruned. Adding a new tool that needs a category simply re-adds its
// constant here and a DefaultLimits entry for it — until then, an unknown or
// empty category falls back to CategoryGeneral and the Default limit.
//
// The seven first-class tools named in Units 32.9 map here as: security_scan →
// CategorySecurity, credential_vend → CategoryCredential, policy_check →
// CategoryPolicy, env_info → CategoryEnvironment, nix_run → CategoryProcess,
// gdev_doctor → CategoryDiagnostics, gdev_status → CategoryStatus.
const (
	CategorySearch      = "search"      // code/text search
	CategorySecurity    = "security"    // vulnerability scanning (security_scan)
	CategoryCredential  = "credential"  // credential vending (credential_vend)
	CategoryPolicy      = "policy"      // policy evaluation (policy_check)
	CategoryEnvironment = "environment" // env probing (env_info)
	CategoryProcess     = "process"     // process execution (nix_run)
	CategoryDiagnostics = "diagnostics" // health checks (gdev_doctor)
	CategoryStatus      = "status"      // state/drift (gdev_status)
	CategoryNetwork     = "network"     // outbound HTTP/API
	CategoryGeneral     = "general"     // uncategorized / default
)

// Limit is the per-category rate-limit and concurrency configuration.
type Limit struct {
	// Rate is the steady-state token refill rate in tokens per second.
	Rate float64
	// Burst is the bucket capacity (the maximum tokens, i.e. the burst size).
	Burst int
	// Concurrency is the maximum number of in-flight calls allowed for this
	// category process-wide (a semaphore, not keyed by agent). Zero or negative
	// means "no concurrency limit".
	Concurrency int
}

// CategoryLimits holds the per-category Limit table plus the Default applied to
// any category not present in Limits (including CategoryGeneral and unknown
// categories).
type CategoryLimits struct {
	Limits  map[string]Limit
	Default Limit
}

// DefaultLimits returns the built-in per-category rate-limit table.
//
// These numbers are principled defaults (the spike that would have pinned them
// is absent — see the categories doc above). The guiding shape: cheap, cached,
// read-only fast-path categories (status, policy, search) get generous
// rate/burst and high concurrency; expensive or externally-facing categories
// (network, security, process, credential) get tighter buckets and limited
// concurrency. CategoryProcess concurrency is fixed at 3 to honor the one
// concrete spec value (nix_run: "max 3 concurrent executions"). Credential
// vending is the most restrictive. The table holds an entry only for an assigned
// category; any category absent here (including CategoryGeneral and unknown
// categories) gets Default.
func DefaultLimits() CategoryLimits {
	return CategoryLimits{
		Default: Limit{Rate: 10, Burst: 20, Concurrency: 8},
		Limits: map[string]Limit{
			CategorySearch:      {Rate: 50, Burst: 100, Concurrency: 16},
			CategorySecurity:    {Rate: 2, Burst: 5, Concurrency: 3},
			CategoryCredential:  {Rate: 0.5, Burst: 5, Concurrency: 2},
			CategoryPolicy:      {Rate: 50, Burst: 100, Concurrency: 16},
			CategoryEnvironment: {Rate: 20, Burst: 40, Concurrency: 8},
			CategoryProcess:     {Rate: 3, Burst: 6, Concurrency: 3},
			CategoryDiagnostics: {Rate: 5, Burst: 10, Concurrency: 4},
			CategoryStatus:      {Rate: 50, Burst: 100, Concurrency: 16},
			CategoryNetwork:     {Rate: 5, Burst: 10, Concurrency: 6},
			CategoryGeneral:     {Rate: 10, Burst: 20, Concurrency: 8},
		},
	}
}

// limitFor returns the Limit for category, falling back to Default when the
// category is absent (covers unknown and empty categories).
func (cl CategoryLimits) limitFor(category string) Limit {
	if l, ok := cl.Limits[category]; ok {
		return l
	}
	return cl.Default
}
