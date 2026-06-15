#!/usr/bin/env bash
# Standalone test harness for lsp-first-guard.sh.
#
# Pipes sample PreToolUse JSON payloads through the hook and asserts the
# resulting verdict (block / warn / allow). Exits non-zero if any case fails.
#
#   bash addons/claudecode/templates/hooks/lsp-first-guard_test.sh

set -u

# Locate the hook relative to this script so the harness is runnable from any
# working directory.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HOOK="$SCRIPT_DIR/lsp-first-guard.sh"

if [ ! -f "$HOOK" ]; then
	printf 'FATAL: hook not found at %s\n' "$HOOK" >&2
	exit 2
fi

fail_count=0
pass_count=0

# classify VERDICT from the hook's stdout: BLOCK | WARN | ALLOW.
classify() {
	local out="$1"
	if [ -z "$out" ]; then
		printf 'ALLOW'
	elif printf '%s' "$out" | grep -q '"permissionDecision"[[:space:]]*:[[:space:]]*"deny"'; then
		printf 'BLOCK'
	elif printf '%s' "$out" | grep -q '"additionalContext"'; then
		printf 'WARN'
	else
		printf 'OTHER'
	fi
}

# assert NAME WANT TIER JSON
assert() {
	local name="$1" want="$2" tier="$3" json="$4"
	local out got
	out="$(printf '%s' "$json" | env QSDEV_LSP_ENFORCEMENT="$tier" bash "$HOOK")"
	got="$(classify "$out")"
	if [ "$got" = "$want" ]; then
		pass_count=$((pass_count + 1))
		printf 'PASS  %-26s [%-5s] => %s\n' "$name" "$tier" "$got"
	else
		fail_count=$((fail_count + 1))
		printf 'FAIL  %-26s [%-5s] => got %s, want %s\n' "$name" "$tier" "$got" "$want" >&2
		printf '      payload: %s\n' "$json" >&2
		printf '      output:  %s\n' "$out" >&2
	fi
}

# grep-helper to build a Grep payload with an optional path/glob.
grep_payload() {
	# args: pattern [path] [glob]
	local pattern="$1" path="${2:-}" glob="${3:-}"
	jq -cn --arg p "$pattern" --arg path "$path" --arg glob "$glob" '
		{tool_name: "Grep", tool_input: (
			{pattern: $p}
			+ (if $path != "" then {path: $path} else {} end)
			+ (if $glob != "" then {glob: $glob} else {} end)
		)}
	'
}

# --- The 10 mandated cases (default block tier). ----------------------------
assert "getUserById"          BLOCK block "$(grep_payload getUserById)"
assert "UserService"          BLOCK block "$(grep_payload UserService)"
assert "router.refresh"       BLOCK block "$(grep_payload router.refresh)"
assert "handle_user_request"  BLOCK block "$(grep_payload handle_user_request)"
assert "TODO"                 ALLOW block "$(grep_payload TODO)"
assert "err"                  ALLOW block "$(grep_payload err)"
assert "API_KEY"              ALLOW block "$(grep_payload API_KEY)"
assert "path docs/"           ALLOW block "$(grep_payload getUserById 'docs/x')"
assert "glob *.md"            ALLOW block "$(grep_payload getUserById '' '*.md')"
assert "quoted string"        ALLOW block "$(grep_payload '"some string"')"

# --- Enforcement-tier cases. ------------------------------------------------
assert "off => allow"         ALLOW off  "$(grep_payload getUserById)"
assert "warn => additional"   WARN  warn "$(grep_payload getUserById)"

# --- Extra robustness cases (fail-open + non-Grep + exemptions). ------------
assert "non-Grep tool"        ALLOW block '{"tool_name":"Read","tool_input":{"pattern":"getUserById"}}'
assert "empty stdin"          ALLOW block ''
assert "not JSON"             ALLOW block 'this is not json at all'
assert "short pattern"        ALLOW block "$(grep_payload abc)"
assert "vendor/ path"         ALLOW block "$(grep_payload UserService 'vendor/foo')"
assert ".sql glob"            ALLOW block "$(grep_payload UserService '' '*.sql')"
assert "regex metachars"      ALLOW block "$(grep_payload 'foo.*bar')"
assert "URL"                  ALLOW block "$(grep_payload 'https://x')"

printf '\n%d passed, %d failed\n' "$pass_count" "$fail_count"
if [ "$fail_count" -ne 0 ]; then
	exit 1
fi
