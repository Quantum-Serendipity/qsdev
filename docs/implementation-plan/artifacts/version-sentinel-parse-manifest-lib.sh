#!/usr/bin/env bash
# Source: https://github.com/KSEGIT/Version-Sentinel/blob/main/scripts/lib/parse-manifest.sh
# Retrieved: 2026-05-12
#
# parse-manifest.sh — per-ecosystem manifest parsers.
# Each parser prints TAB-separated "pkg\tversion" lines, one per dependency.
# Version prefixes (^ ~ >= <= = v) are stripped.
# Local/git/workspace refs are skipped.
# Missing/invalid file -> empty output, exit 0 (fail-open).

_strip_version_prefix() {
  sed -E 's/^[v^~><= ]+//' <<< "$1"
}

_is_registry_version() {
  local raw="$1"
  case "$raw" in
    file:*|git+*|git:*|github:*|workspace:*|link:*|portal:*|npm:*|"*"|""|latest|next) return 1 ;;
  esac
  return 0
}

parse_npm() {
  local file="$1"
  [[ -f "$file" ]] || return 0
  jq -r '[.dependencies, .devDependencies, .peerDependencies, .optionalDependencies]
         | map(select(. != null)) | add // {} | to_entries[] | "\(.key)\t\(.value)"' \
    "$file" 2>/dev/null | while IFS=$'\t' read -r pkg raw; do
      [[ -z "$pkg" ]] && continue
      _is_registry_version "$raw" || continue
      local ver
      ver=$(_strip_version_prefix "$raw")
      [[ "$ver" =~ [[:space:]] ]] && continue
      printf '%s\t%s\n' "$pkg" "$ver"
    done
}

parse_pip() {
  local file="$1"
  [[ -f "$file" ]] || return 0
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%%#*}"
    line="${line%"${line##*[![:space:]]}"}"
    [[ -z "$line" ]] && continue
    case "$line" in
      -*|*://*|./*|../*|/*) continue ;;
    esac
    line="${line%%;*}"
    line="${line%"${line##*[![:space:]]}"}"
    [[ "$line" == *@* && "$line" != *==* ]] && continue
    if [[ "$line" =~ ^([A-Za-z0-9][A-Za-z0-9._-]*)[[:space:]]*(==|~=|\>=|\<=|\>|\<|!=)[[:space:]]*([A-Za-z0-9][A-Za-z0-9._*+-]*) ]]; then
      local pkg="${BASH_REMATCH[1]}" ver="${BASH_REMATCH[3]}"
      printf '%s\t%s\n' "$pkg" "$ver"
    fi
  done < "$file"
}

# ... (pyproject, cargo, csproj parsers follow the same pattern)

ecosystem_for_path() {
  local path="$1"
  local base
  base=$(basename "$path")
  case "$base" in
    package.json) echo "npm" ;;
    requirements*.txt|constraints*.txt) echo "pip" ;;
    pyproject.toml) echo "pyproject" ;;
    Cargo.toml) echo "cargo" ;;
    *.csproj|*.fsproj|*.vbproj) echo "csproj" ;;
    *) echo "" ;;
  esac
}

parse_manifest_by_path() {
  local path="$1"
  local eco
  eco=$(ecosystem_for_path "$path")
  case "$eco" in
    npm) parse_npm "$path" ;;
    pip) parse_pip "$path" ;;
    pyproject) parse_pyproject "$path" ;;
    cargo) parse_cargo "$path" ;;
    csproj) parse_csproj "$path" ;;
    *) return 0 ;;
  esac
}

diff_manifest_sets() {
  local pre="$1" post="$2"
  local tmp_pre tmp_post
  tmp_pre=$(mktemp); tmp_post=$(mktemp)
  printf '%s\n' "$pre" | sort -u > "$tmp_pre"
  printf '%s\n' "$post" | sort -u > "$tmp_post"
  while IFS=$'\t' read -r pkg ver; do
    [[ -z "$pkg" ]] && continue
    local pre_ver
    pre_ver=$(awk -F '\t' -v p="$pkg" '$1==p {print $2; exit}' "$tmp_pre")
    if [[ -z "$pre_ver" ]]; then
      printf 'added\t%s\t%s\n' "$pkg" "$ver"
    elif [[ "$pre_ver" != "$ver" ]]; then
      printf 'changed\t%s\t%s\n' "$pkg" "$ver"
    fi
  done < "$tmp_post"
  rm -f "$tmp_pre" "$tmp_post"
}
