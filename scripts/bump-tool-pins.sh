#!/usr/bin/env bash
# bump-tool-pins.sh: move the pins in the CI tool pin files to the newest
# exact release that has aged past the cooldown (U26-02, Decision D9).
#
# Usage: scripts/bump-tool-pins.sh [--open-pr] [ENV_FILE...]
#
# With no ENV_FILE it bumps the repository's three pin files together:
# .github/tool-versions.env, the CI pin catalog's Nix release pins
# (internal/cigeneration/nix-release.env) and their generated copy
# (.github/nix-release.env), so the catalog and its copy never drift. Each
# KEY=value line in an ENV_FILE must sit directly below a "# source:" line:
#   # source: github <owner>/<repo>       GitHub releases (gh api)
#   # source: goproxy <module>            versions on proxy.golang.org
#   # source: nix-release <owner>/<repo>  Nix X.Y.Z tags (gh api) whose
#                                         tarballs releases.nixos.org serves
#   # source: nix-sha256 <system>         digest of that release's tarball
# A pin moves to the newest exact vX.Y.Z (X.Y.Z for Nix) release newer than it
# that was published at least MIN_AGE_DAYS ago, even when a younger release is
# newer still; it never moves backwards. Drafts, prereleases and non-exact tags
# are ignored. For goproxy the age is the version's commit time, which is never
# later than its tag; for Nix it is the Last-Modified time of the release's
# published x86_64-linux .sha256 on releases.nixos.org, and a tag with no
# published tarball is not a release yet. When NIX_VERSION moves, every
# nix-sha256 line below it is replaced with the new release's published
# digest; .github/scripts/install-nix.sh checks the tarball against it in CI.
#
# Each upstream release list and digest is read and aged once per run, so a
# release pinned in several files moves them all to the same version. Every
# file's new content is computed before any file is written: malformed or
# missing upstream data fails the run with every file untouched.
#
# --open-pr commits every changed file to the tool-pins/bump branch in one
# commit, force-pushes it and opens a PR unless one is already open. It needs
# GH_TOKEN with contents:write and pull_requests:write; tool-pins.yml passes a
# PAT so the PR triggers CI.
set -euo pipefail

# The catalog's baseline age gate (72h in internal/catalog/defaults.yaml).
# TestToolPinsWorkflowScope fails if this drops below it.
readonly MIN_AGE_DAYS=3
readonly PR_BRANCH="tool-pins/bump"
readonly NIX_RELEASES_URL="https://releases.nixos.org/nix"

open_pr=false
if [ "${1:-}" = "--open-pr" ]; then
  open_pr=true
  shift
fi
if [ $# -eq 0 ]; then
  set -- .github/tool-versions.env internal/cigeneration/nix-release.env .github/nix-release.env
fi
env_files=("$@")

die() {
  echo "bump-tool-pins: $*" >&2
  exit 1
}

# releases KIND TARGET CURRENT prints "<version> <published time>" for the
# target's releases; for goproxy only the exact versions newer than CURRENT,
# since each needs its own request. The caller filters and ages them.
releases() {
  case "$1" in
    github)
      # The newest 100 releases; an aged one newer than the pin is among them.
      gh api "repos/$2/releases?per_page=100" |
        jq -r 'if type == "array" then .[] else error("not a release list") end
          | select(.draft == false and .prerelease == false)
          | "\(.tag_name) \(.published_at)"'
      ;;
    goproxy)
      # Module paths here are lower-case, so they need no proxy escaping.
      local list v
      list="$(curl -fsSL --proto '=https' "https://proxy.golang.org/$2/@v/list")" || return 1
      for v in $list; do
        { [[ "$v" =~ $exact_re ]] && newer "$v" "$3"; } || continue
        curl -fsSL --proto '=https' "https://proxy.golang.org/$2/@v/$v.info" |
          jq -er '"\(.Version) \(.Time)"' || return 1
      done
      ;;
    nix-release)
      # Nix publishes no GitHub releases, only tags; the newest 100 tags hold
      # any aged release newer than the pin.
      local tags v published rc
      tags="$(gh api "repos/$2/tags?per_page=100" |
        jq -r 'if type == "array" then .[].name else error("not a tag list") end')" || return 1
      for v in $tags; do
        { [[ "$v" =~ $nix_re ]] && newer "$v" "$3"; } || continue
        rc=0
        published="$(nix_published "$v")" || rc=$?
        case "$rc" in
          0) echo "$v $published" ;;
          2) ;; # tagged, but no tarball published yet
          *) return 1 ;;
        esac
      done
      ;;
    *)
      return 1
      ;;
  esac
}

# age_days TIME prints whole days since an RFC 3339 UTC time.
age_days() {
  jq -nr --arg t "$1" '(now - ($t | sub("\\.[0-9]+Z$"; "Z") | fromdateiso8601)) / 86400 | floor'
}

# newer A B succeeds when vX.Y.Z A is strictly newer than B.
newer() {
  jq -ne --arg a "$1" --arg b "$2" \
    'def parts: ltrimstr("v") | split(".") | map(tonumber); ($a | parts) > ($b | parts)' >/dev/null
}

# nix_published VERSION prints the RFC 3339 time releases.nixos.org published
# VERSION's x86_64-linux tarball digest. It returns 2 when the server answers
# with an HTTP error (the tag has no published release yet) and 1 on any other
# failure.
nix_published() {
  local headers line modified=""
  headers="$(curl -fsSI --proto '=https' "$NIX_RELEASES_URL/nix-$1/nix-$1-x86_64-linux.tar.xz.sha256")" ||
    { [ $? -eq 22 ] && return 2; return 1; }
  while IFS= read -r line; do
    line="${line%$'\r'}"
    if [[ "$line" =~ ^[Ll][Aa][Ss][Tt]-[Mm][Oo][Dd][Ii][Ff][Ii][Ee][Dd]:\ *(.*)$ ]]; then
      modified="${BASH_REMATCH[1]}"
    fi
  done <<<"$headers"
  [ -n "$modified" ] || return 1
  # An unreadable date prints the raw value, which age_days then rejects.
  jq -nr --arg t "$modified" '$t | (strptime("%a, %d %b %Y %H:%M:%S GMT") | todateiso8601)? // $t'
}

# nix_sha256 VERSION SYSTEM prints the published SHA-256 of VERSION's tarball
# for SYSTEM.
nix_sha256() {
  local digest
  digest="$(curl -fsSL --proto '=https' "$NIX_RELEASES_URL/nix-$1/nix-$1-$2.tar.xz.sha256")" || return 1
  digest="${digest%%[[:space:]]*}"
  [[ "$digest" =~ ^[0-9a-f]{64}$ ]] || return 1
  echo "$digest"
}

exact_re='^v[0-9]+\.[0-9]+\.[0-9]+$'
nix_re='^[0-9]+\.[0-9]+\.[0-9]+$'

# The per-run upstream cache. Entry i of upstream_keys ("<kind> <target>")
# holds, in upstream_lists, "<version> <published> <age days>" for every
# release newer than upstream_floors[i]; it answers any pin at or above that
# floor. Indexed arrays keep it to plain bash.
upstream_keys=()
upstream_floors=()
upstream_lists=()
digest_keys=()
digest_values=()

# upstream KEY KIND TARGET CURRENT sets upstream_list to "<version>
# <published> <age days>" lines covering every exact release of TARGET newer
# than CURRENT (it may hold older ones too; the caller filters). A release
# list is read and aged once per run. KEY names the pin in errors.
upstream() {
  local key="$1" kind="$2" target="$3" current="$4" i=0 re list version published age
  while [ "$i" -lt "${#upstream_keys[@]}" ]; do
    if [ "${upstream_keys[$i]}" = "$kind $target" ] && ! newer "${upstream_floors[$i]}" "$current"; then
      upstream_list="${upstream_lists[$i]}"
      return 0
    fi
    i=$((i + 1))
  done

  re="$exact_re"
  [ "$kind" != nix-release ] || re="$nix_re"
  list="$(releases "$kind" "$target" "$current")" || die "$key: cannot read the releases of $kind $target"
  upstream_list=""
  while read -r version published; do
    { [ -n "$version" ] && [[ "$version" =~ $re ]] && newer "$version" "$current"; } || continue
    age="$(age_days "$published")" || die "$key: $kind $target reports an unreadable time for $version: $published"
    upstream_list+="$version $published $age"$'\n'
  done <<<"$list"
  upstream_keys+=("$kind $target")
  upstream_floors+=("$current")
  upstream_lists+=("$upstream_list")
}

# cached_nix_sha256 KEY VERSION SYSTEM sets digest to the published SHA-256 of
# VERSION's tarball for SYSTEM, reading each one once per run.
cached_nix_sha256() {
  local i=0
  while [ "$i" -lt "${#digest_keys[@]}" ]; do
    if [ "${digest_keys[$i]}" = "$2 $3" ]; then
      digest="${digest_values[$i]}"
      return 0
    fi
    i=$((i + 1))
  done
  digest="$(nix_sha256 "$2" "$3")" || die "$1: cannot read the published SHA-256 of nix $2 for $3"
  digest_keys+=("$2 $3")
  digest_values+=("$digest")
}

# pick_release KEY KIND TARGET CURRENT sets best, best_time and best_age to
# the newest release of TARGET newer than CURRENT that has aged MIN_AGE_DAYS,
# or best to "" when there is none, and reports a newer one still held back.
pick_release() {
  local key="$1" kind="$2" target="$3" current="$4" held="" version published age
  upstream "$key" "$kind" "$target" "$current"
  best=""
  best_time=""
  best_age=""
  while read -r version published age; do
    { [ -n "$version" ] && newer "$version" "$current"; } || continue
    if [ "$age" -lt "$MIN_AGE_DAYS" ]; then
      if [ -z "$held" ] || newer "$version" "$held"; then held="$version"; fi
    elif [ -z "$best" ] || newer "$version" "$best"; then
      best="$version"
      best_time="$published"
      best_age="$age"
    fi
  done <<<"$upstream_list"

  if [ -n "$held" ] && { [ -z "$best" ] || newer "$held" "$best"; }; then
    echo "$key: $held is under the $MIN_AGE_DAYS-day cooldown"
  fi
}

# bump_file FILE sets file_out to FILE's content with every pin moved and
# file_summary to one "- KEY: ..." line per moved pin ("" when none moved).
# It writes nothing.
bump_file() {
  local line key current kind target source_line="" nix_version="" nix_moved=false
  file_out=""
  file_summary=""
  while IFS= read -r line || [ -n "$line" ]; do
    if [[ "$line" =~ ^([A-Z][A-Z0-9_]*)=(.*)$ ]]; then
      key="${BASH_REMATCH[1]}"
      current="${BASH_REMATCH[2]}"
      [[ "$source_line" =~ ^#\ source:\ (github|goproxy|nix-release|nix-sha256)\ ([^[:space:]]+)$ ]] ||
        die "$key: no valid \"# source: github|goproxy|nix-release|nix-sha256 <target>\" line above it"
      kind="${BASH_REMATCH[1]}"
      target="${BASH_REMATCH[2]}"

      if [ "$kind" = nix-sha256 ]; then
        [ -n "$nix_version" ] || die "$key: a nix-sha256 line must follow the nix-release line"
        if [ "$nix_moved" = true ]; then
          cached_nix_sha256 "$key" "$nix_version" "$target"
          line="$key=$digest"
          file_summary+="- $key: digest of nix-$nix_version-$target.tar.xz"$'\n'
        fi
      else
        if [ "$kind" = nix-release ]; then
          [[ "$current" =~ $nix_re ]] || die "$key=$current is not an exact Nix release (want X.Y.Z)"
          nix_version="$current"
        fi
        pick_release "$key" "$kind" "$target" "$current"
        if [ -z "$best" ]; then
          echo "$key: keeping $current; no newer release has aged $MIN_AGE_DAYS days"
        else
          echo "$key: $current -> $best ($best_age days old)"
          line="$key=$best"
          file_summary+="- $key: $current -> $best ($kind $target, published $best_time)"$'\n'
          if [ "$kind" = nix-release ]; then
            nix_version="$best"
            nix_moved=true
          fi
        fi
      fi
    fi
    file_out+="$line"$'\n'
    source_line="$line"
  done <"$1"
}

for f in "${env_files[@]}"; do
  [ -f "$f" ] || die "$f not found"
done

# Compute every file before writing any.
new_contents=()
summaries=()
for f in "${env_files[@]}"; do
  echo "$f:"
  bump_file "$f"
  new_contents+=("$file_out")
  summaries+=("$file_summary")
done

changed=()
summary=""
i=0
while [ "$i" -lt "${#env_files[@]}" ]; do
  if [ -n "${summaries[$i]}" ]; then
    f="${env_files[$i]}"
    tmp="$(mktemp "${f}.XXXXXX")"
    printf '%s' "${new_contents[$i]}" >"$tmp"
    mv "$tmp" "$f"
    changed+=("$f")
    summary+="$f:"$'\n'"${summaries[$i]}"
  fi
  i=$((i + 1))
done

if [ -z "$summary" ]; then
  echo "All tool pins are current."
  exit 0
fi

if [ "$open_pr" != true ]; then
  exit 0
fi

title="chore(deps): bump CI tool pins"
body="Bumped by .github/workflows/tool-pins.yml. Each release is at least ${MIN_AGE_DAYS} days old."$'\n\n'"$summary"
git config user.name "qsdev-tool-pins"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git switch -C "$PR_BRANCH"
git commit -m "$title" -m "$summary" -- "${changed[@]}"
git push --force origin "HEAD:refs/heads/$PR_BRANCH"
if [ -z "$(gh pr list --head "$PR_BRANCH" --state open --json number --jq '.[].number')" ]; then
  gh pr create --head "$PR_BRANCH" --title "$title" --body "$body"
else
  echo "Updated the open PR on $PR_BRANCH."
fi
