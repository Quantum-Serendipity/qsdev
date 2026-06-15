#!/usr/bin/env bash
# lsp-first-guard — PreToolUse hook on the Grep tool (qsdev Phase 31).
#
# Redirects code-symbol searches to Claude Code's first-class LSP tool
# (goToDefinition / findReferences / workspaceSymbol / hover / ...) for
# precise, token-efficient navigation. Plain-text / literal / config-file
# searches pass straight through to Grep.
#
# Enforcement tiers (QSDEV_LSP_ENFORCEMENT):
#   block (default) — deny the Grep and tell the model to use the LSP tool.
#   warn            — allow the Grep but inject guidance (additionalContext).
#   off             — disabled; pass everything through.
#
# This is a productivity hook, NOT a security control: it FAILS OPEN. Any
# parser error, empty/non-JSON stdin, or missing field results in exit 0
# (allow). It never blocks Grep on its own malfunction.

set -euo pipefail

# --- a. Read stdin; fail open on empty or unparseable input. ----------------
input="$(cat)"
if [ -z "$input" ]; then
	exit 0
fi

# Extract fields in one jq pass. If jq fails (not installed, invalid JSON),
# fall through to exit 0 via the `|| exit 0` guard.
parsed="$(printf '%s' "$input" | jq -r '
	[ (.tool_name // ""),
	  (.tool_input.pattern // ""),
	  (.tool_input.path // ""),
	  (.tool_input.glob // "") ]
	| @tsv
' 2>/dev/null)" || exit 0

if [ -z "$parsed" ]; then
	exit 0
fi

# Split the tab-separated record into fields.
IFS=$'\t' read -r tool_name pattern path glob <<<"$parsed"

# --- b. Honor the enforcement tier. -----------------------------------------
tier="${QSDEV_LSP_ENFORCEMENT:-block}"
if [ "$tier" = "off" ]; then
	exit 0
fi

# --- c. Only act on the Grep tool. ------------------------------------------
if [ "$tool_name" != "Grep" ]; then
	exit 0
fi

# --- d. Ignore short / empty patterns. --------------------------------------
if [ -z "$pattern" ] || [ "${#pattern}" -lt 4 ]; then
	exit 0
fi

# --- e. Path / glob exemptions. ---------------------------------------------
# Searches scoped to non-code locations or restricted to non-code file types
# are legitimate text searches — pass them through.
scope="$path $glob"

case "$scope" in
*docs/* | *.claude/* | *node_modules/* | *vendor/* | *dist/* | *build/*)
	exit 0
	;;
esac

# Non-code extensions on the path or glob (documentation, config, data, etc.).
for ext in .md .markdown .json .yaml .yml .toml .env .csv .tsv .xml .sql \
	.sh .bash .css .scss .less .html .txt .lock .ini .cfg; do
	case "$path" in *"$ext") exit 0 ;; esac
	case "$glob" in *"$ext") exit 0 ;; esac
done

# --- f. Allowlist: patterns that are clearly NOT a single code symbol. ------
# Dotted access (identifier.identifier, e.g. router.refresh) is a code symbol,
# not a regex search. Recognize it up front so the generic metacharacter
# allowlist below does not swallow it on account of the literal '.'.
is_dotted_access=false
if [[ "$pattern" =~ ^[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*$ ]]; then
	is_dotted_access=true
fi

# Regex metacharacters → the user is doing a real regex search, not a symbol
# lookup. Any of: \ ^ $ . | ? * + ( ) [ ] { }  (dotted access is exempt).
if [ "$is_dotted_access" != true ]; then
	case "$pattern" in
	*[\\^\$.\|?*+\(\)\[\]\{\}]*)
		exit 0
		;;
	esac
fi

# Log / diagnostic markers and common logging call prefixes.
case "$pattern" in
TODO* | FIXME* | HACK* | XXX* | NOTE* | WARN* | ERROR* | DEBUG* | INFO*)
	exit 0
	;;
esac

# Quoted string literals (start and end with matching quote). Note: quotes are
# not in the metachar set above, so these reach here intact.
case "$pattern" in
'"'*'"' | "'"*"'")
	exit 0
	;;
esac

# Remaining allowlist forms need regex matching (no disallowed metachars
# survive to here, so these are safe, deterministic structural checks).
if [[ "$pattern" =~ ^(import|require|export|use|include)$ ]] ||
	[[ "$pattern" =~ ^https?:// ]] ||
	[[ "$pattern" =~ ^[A-Z][A-Z0-9_]*$ ]] ||
	[[ "$pattern" =~ ^[a-z]{1,8}$ ]]; then
	exit 0
fi

# --- g. Symbol detection. ---------------------------------------------------
is_symbol=false

# camelCase: lower-led, has an internal capital, length >= 4.
if [ "${#pattern}" -ge 4 ] && [[ "$pattern" =~ ^[a-z][a-zA-Z0-9]*[A-Z][a-zA-Z0-9]*$ ]]; then
	is_symbol=true
fi

# PascalCase: upper-led, lower/digit second char, length >= 4.
if [ "${#pattern}" -ge 4 ] && [[ "$pattern" =~ ^[A-Z][a-z0-9][a-zA-Z0-9]*$ ]]; then
	is_symbol=true
fi

# Dotted access: identifier.identifier (e.g. router.refresh).
if [ "$is_dotted_access" = true ]; then
	is_symbol=true
fi

# Long snake_case: a multi-word lower_snake identifier (>= 2 underscores and
# length >= 10, e.g. handle_user_request).
underscores="${pattern//[^_]/}"
if [ "${#underscores}" -ge 2 ] && [ "${#pattern}" -ge 10 ]; then
	is_symbol=true
fi

if [ "$is_symbol" != true ]; then
	exit 0
fi

# --- Emit the decision. -----------------------------------------------------
reason="LSP-FIRST: pattern '${pattern}' looks like a code symbol. Use the LSP tool (goToDefinition / findReferences / workspaceSymbol) instead of Grep for precise, token-efficient results."

if [ "$tier" = "warn" ]; then
	# Allow the Grep but inject guidance (omit permissionDecision).
	jq -cn --arg ctx "$reason" \
		'{hookSpecificOutput: {hookEventName: "PreToolUse", additionalContext: $ctx}}'
	exit 0
fi

# Default (block): deny the Grep and steer the model to the LSP tool.
jq -cn --arg reason "$reason" \
	'{hookSpecificOutput: {hookEventName: "PreToolUse", permissionDecision: "deny", permissionDecisionReason: $reason}}'
exit 0
