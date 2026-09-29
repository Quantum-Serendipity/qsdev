package ecosystem

import "strings"

// LicensePolicyPath is the project-relative ScanCode license policy file the
// license-compliance tool generates. ScanCode reads no project config file of
// its own, so the security-scan task passes it with --license-policy.
const LicensePolicyPath = ".scancode.yml"

// LicenseExceptionsPath is the project-relative file in which teams record
// approved license exceptions. The license-compliance tool creates it once and
// never overwrites it.
const LicenseExceptionsPath = ".license-exceptions.yml"

// licenseScanIgnores are the --ignore patterns of the license scan. ScanCode
// matches a pattern without a slash against every path segment, so each entry
// skips that directory (or file) anywhere in the tree.
//
// Unlike the SAST scan exclusions, the dependency directories (node_modules,
// vendor, third_party, virtual environments) are deliberately scanned: the
// license texts and manifests of the project's dependencies live there, and
// they are what the policy exists to check. What is skipped is build output,
// caches, generated framework output, the devenv state, test fixtures and the
// policy files themselves, whose license names would otherwise be reported as
// license findings.
var licenseScanIgnores = []string{
	".git",
	LicensePolicyPath,
	LicenseExceptionsPath,

	// Build output
	"build",
	"dist",
	"out",
	"result",
	"target",

	// Caches
	".cache",
	".pytest_cache",
	"__pycache__",

	// Coverage
	".coverage",
	"coverage",

	// Framework output
	".next",
	".nuxt",

	// Nix / devenv
	".devenv",
	".direnv",

	// Test fixtures
	"fixtures",
	"testdata",

	// Packaging metadata and tox environments
	"*.egg-info",
	".tox",
}

// LicenseScanIgnores returns a copy of the path patterns the license scan
// skips.
func LicenseScanIgnores() []string {
	return append([]string(nil), licenseScanIgnores...)
}

// licensePolicyGate is the jq program that turns the ScanCode JSON scan into a
// pass/fail result. ScanCode's --license-policy only annotates each file with
// the policy entries its detected licenses match (the "license_policy" list);
// it never fails the scan, so the gate does:
//   - an entry whose compliance_alert is "warning" is printed for manual
//     review and does not fail the task;
//   - any entry whose compliance_alert is "error" is printed to stderr and
//     makes jq exit 1, which fails the task under pipefail;
//   - any error ScanCode records in the scan headers fails the task too. A
//     policy file ScanCode rejects (empty, or with a duplicate license_key)
//     is reported only there: ScanCode then applies no policy and still exits
//     0, so without this check a broken policy would pass every scan.
//
// Older ScanCode releases emitted license_policy as a single object rather
// than a list, so both shapes are accepted.
const licensePolicyGate = `[(.files // [])[] | .path as $p` +
	` | (.license_policy // []) | (if type == "object" then [.] else . end)[]` +
	` | {path: $p, license: (.spdx_license_key // .license_key), label: .label, alert: .compliance_alert}] as $hits` +
	` | ($hits[] | select(.alert == "warning") | "license needs review (\(.label)): \(.path): \(.license)")` +
	`, ([$hits[] | select(.alert == "error") | "prohibited license (\(.label)): \(.path): \(.license)"]` +
	` | if length > 0 then (join("\n") + "\n") | halt_error(1) else empty end)` +
	`, ([(.headers // [])[] | (.errors // [])[]]` +
	` | if length > 0 then ("scancode reported errors:\n" + join("\n") + "\n") | halt_error(1) else empty end)`

// licenseScanCommand runs a ScanCode license scan of the project with the
// generated policy applied, writes the JSON result to stdout and pipes it
// through licensePolicyGate, so a file carrying a prohibited license fails the
// security-scan task. ScanCode picks its own parallelism (CPUs - 1).
func licenseScanCommand() string {
	var b strings.Builder
	b.WriteString("scancode --quiet --license --license-policy ")
	b.WriteString(LicensePolicyPath)
	for _, p := range licenseScanIgnores {
		b.WriteString(" --ignore '")
		b.WriteString(p)
		b.WriteString("'")
	}
	b.WriteString(" --json - . | jq -r '")
	b.WriteString(licensePolicyGate)
	b.WriteString("'")
	return b.String()
}
