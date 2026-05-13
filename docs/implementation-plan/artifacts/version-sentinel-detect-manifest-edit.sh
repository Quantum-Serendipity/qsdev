#!/usr/bin/env bash
set -u

DIR="$(dirname "$0")"
# shellcheck source=lib/options.sh
source "$DIR/lib/options.sh"

if [[ "${VS_DISABLE:-0}" == "1" ]]; then exit 0; fi

# shellcheck source=lib/parse-manifest.sh
source "$DIR/lib/parse-manifest.sh"
# shellcheck source=lib/sidecar.sh
source "$DIR/lib/sidecar.sh"

input=$(cat)

if ! command -v jq >/dev/null 2>&1; then
  echo "version-sentinel: jq missing, fail-open" >&2; exit 0
fi
if ! echo "$input" | jq -e . >/dev/null 2>&1; then
  echo "version-sentinel: unparseable tool_input JSON, fail-open" >&2; exit 0
fi

tool_name=$(echo "$input" | jq -r '.tool_name // empty')
file_path=$(echo "$input" | jq -r '.tool_input.file_path // empty')
[[ -z "$file_path" ]] && exit 0

eco=$(ecosystem_for_path "$file_path")
[[ -z "$eco" ]] && exit 0

pre_content=""
[[ -f "$file_path" ]] && pre_content=$(cat "$file_path")
post_content="$pre_content"

# py_replace_once <old> <new> <content-on-stdin> — replaces first occurrence only.
# Uses python3 for bash-3.2 compatibility (macOS default).
py_replace_once() {
  python3 -c 'import sys; p=sys.stdin.read(); sys.stdout.write(p.replace(sys.argv[1], sys.argv[2], 1))' "$1" "$2"
}

case "$tool_name" in
  Edit)
    old=$(echo "$input" | jq -r '.tool_input.old_string // empty' | tr -d '\r')
    new=$(echo "$input" | jq -r '.tool_input.new_string // empty' | tr -d '\r')
    replace_all=$(echo "$input" | jq -r '.tool_input.replace_all // false')
    if [[ "$replace_all" == "true" ]]; then
      post_content=$(awk -v o="$old" -v n="$new" 'BEGIN{RS=""} { gsub(o, n); print }' <<< "$pre_content")
    else
      post_content=$(printf '%s' "$pre_content" | py_replace_once "$old" "$new")
    fi
    ;;
  Write)
    post_content=$(echo "$input" | jq -r '.tool_input.content // empty' | tr -d '\r')
    ;;
  MultiEdit)
    post_content="$pre_content"
    edits_tsv=$(echo "$input" | jq -r '.tool_input.edits[]? | [.old_string, .new_string] | @tsv')
    while IFS=$'\t' read -r o n; do
      [[ -z "$o" ]] && continue
      o=$(printf '%s' "$o" | tr -d '\r')
      n=$(printf '%s' "$n" | tr -d '\r')
      post_content=$(printf '%s' "$post_content" | py_replace_once "$o" "$n")
    done <<< "$edits_tsv"
    ;;
  *) exit 0 ;;
esac

# Parse via temp dirs with files named after the real manifest so ecosystem_for_path dispatches correctly
manifest_name=$(basename "$file_path")
tmp_dir_pre=$(mktemp -d)
tmp_dir_post=$(mktemp -d)
tmp_pre="$tmp_dir_pre/$manifest_name"
tmp_post="$tmp_dir_post/$manifest_name"
trap 'rm -rf "$tmp_dir_pre" "$tmp_dir_post"' EXIT

printf '%s' "$pre_content" > "$tmp_pre"
printf '%s' "$post_content" > "$tmp_post"

pre_deps=$(parse_manifest_by_path "$tmp_pre" 2>/dev/null)
post_deps=$(parse_manifest_by_path "$tmp_post" 2>/dev/null)

changes=$(diff_manifest_sets "$pre_deps" "$post_deps")
[[ -z "$changes" ]] && exit 0

block=0
block_msgs=""
while IFS=$'\t' read -r kind pkg ver; do
  [[ -z "$pkg" ]] && continue
  if ! bash "$DIR/check-sidecar.sh" "$eco" "$pkg" "$ver" 2>/tmp/_vs_err_$$; then
    block=1
    block_msgs+=$(cat /tmp/_vs_err_$$)$'\n---\n'
  fi
done <<< "$changes"
rm -f /tmp/_vs_err_$$

if [[ "$block" -eq 1 ]]; then
  echo "$block_msgs" >&2
  exit 2
fi
exit 0
