package middleware

// The 19-category taxonomy.
//
// Unit 32.8 references a "19 categories from the design spike" but the cited
// spike (research-spikes/gdev-universal-mcp-server-design) is not present in the
// repository, so neither the exact category names nor their rate-limit defaults
// are recoverable. The taxonomy below is a principled 19-category set covering
// the tool surface a universal dev-tooling MCP server exposes (the seven tools
// named in Units 32.9 — credential_vend, security_scan, policy_check, env_info,
// nix_run, gdev_status, gdev_doctor — plus the generic dev-tool categories an
// adapter ecosystem needs). Tools are mapped to a category via
// spi.ToolRegistration.Category; an unknown/empty category falls back to
// CategoryGeneral and the default limit.
const (
	CategoryFilesystem       = "filesystem"        // read/write project files
	CategorySearch           = "search"            // code/text search
	CategoryCodeIntelligence = "code_intelligence" // LSP/AST navigation
	CategoryVersionControl   = "version_control"   // git operations
	CategoryBuild            = "build"             // compile/build
	CategoryTest             = "test"              // run tests
	CategoryPackage          = "package"           // dependency management
	CategorySecurity         = "security"          // vulnerability scanning (security_scan)
	CategoryCredential       = "credential"        // credential vending (credential_vend)
	CategoryPolicy           = "policy"            // policy evaluation (policy_check)
	CategoryEnvironment      = "environment"       // env probing (env_info)
	CategoryProcess          = "process"           // process execution (nix_run)
	CategoryDiagnostics      = "diagnostics"       // health checks (gdev_doctor)
	CategoryStatus           = "status"            // state/drift (gdev_status)
	CategoryDocumentation    = "documentation"     // docs lookup
	CategoryNetwork          = "network"           // outbound HTTP/API
	CategoryDatabase         = "database"          // database queries
	CategoryContainer        = "container"         // container/orchestration
	CategoryGeneral          = "general"           // uncategorized / default
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
// is absent — see categories doc above). The guiding shape: cheap, cached,
// read-only fast-path categories (status, policy, search, filesystem, docs) get
// generous rate/burst and high concurrency; expensive or externally-facing
// categories (build, test, network, security, process, credential) get tighter
// buckets and limited concurrency. CategoryProcess concurrency is fixed at 3 to
// honor the one concrete spec value (nix_run: "max 3 concurrent executions").
// Credential vending is the most restrictive.
func DefaultLimits() CategoryLimits {
	return CategoryLimits{
		Default: Limit{Rate: 10, Burst: 20, Concurrency: 8},
		Limits: map[string]Limit{
			CategoryFilesystem:       {Rate: 50, Burst: 100, Concurrency: 16},
			CategorySearch:           {Rate: 50, Burst: 100, Concurrency: 16},
			CategoryCodeIntelligence: {Rate: 30, Burst: 60, Concurrency: 12},
			CategoryVersionControl:   {Rate: 20, Burst: 40, Concurrency: 8},
			CategoryBuild:            {Rate: 2, Burst: 4, Concurrency: 2},
			CategoryTest:             {Rate: 2, Burst: 4, Concurrency: 2},
			CategoryPackage:          {Rate: 5, Burst: 10, Concurrency: 4},
			CategorySecurity:         {Rate: 2, Burst: 5, Concurrency: 3},
			CategoryCredential:       {Rate: 0.5, Burst: 5, Concurrency: 2},
			CategoryPolicy:           {Rate: 50, Burst: 100, Concurrency: 16},
			CategoryEnvironment:      {Rate: 20, Burst: 40, Concurrency: 8},
			CategoryProcess:          {Rate: 3, Burst: 6, Concurrency: 3},
			CategoryDiagnostics:      {Rate: 5, Burst: 10, Concurrency: 4},
			CategoryStatus:           {Rate: 50, Burst: 100, Concurrency: 16},
			CategoryDocumentation:    {Rate: 30, Burst: 60, Concurrency: 12},
			CategoryNetwork:          {Rate: 5, Burst: 10, Concurrency: 6},
			CategoryDatabase:         {Rate: 10, Burst: 20, Concurrency: 8},
			CategoryContainer:        {Rate: 5, Burst: 10, Concurrency: 4},
			CategoryGeneral:          {Rate: 10, Burst: 20, Concurrency: 8},
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
