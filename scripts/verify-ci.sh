#!/usr/bin/env bash
# verify-ci.sh: fail unless every required check passed on a commit (U26-06).
#
# Usage: scripts/verify-ci.sh REPO SHA [CHECKS_FILE]
#
# REPO is owner/name and SHA the full commit hash. CHECKS_FILE (default
# .github/required-checks.txt) lists one check-run name per line; blank lines
# and lines starting with # are ignored. The release workflow runs this
# before anything is built or published, so a tag on a commit that did not
# pass CI releases nothing.
#
# Every listed check must have a latest check run on SHA, and every such run
# must have completed with conclusion success. A missing, queued, in-progress,
# failed, cancelled, timed-out, neutral or skipped run fails the gate and is
# named in the output. Check runs that are not listed are ignored. Needs gh
# (authenticated, checks:read) and jq.
set -euo pipefail

die() {
  echo "::error::verify-ci: $*" >&2
  exit 1
}

[ $# -ge 2 ] && [ $# -le 3 ] || die "usage: verify-ci.sh REPO SHA [CHECKS_FILE]"
repo="$1"
sha="$2"
checks_file="${3:-.github/required-checks.txt}"

[[ "$repo" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || die "REPO $repo is not owner/name"
[[ "$sha" =~ ^([0-9a-f]{40}|[0-9a-f]{64})$ ]] || die "SHA $sha is not a full commit hash"
[ -r "$checks_file" ] || die "cannot read $checks_file"

# The required names, one per line, without comments, blank lines or the \r
# a Windows checkout may leave.
required="$(tr -d '\r' < "$checks_file" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' | grep -v -e '^$' -e '^#' || true)"
[ -n "$required" ] || die "$checks_file lists no checks"

# One "name<TAB>status<TAB>conclusion" line per latest check run. --paginate
# prints one JSON object per page; jq reads them all. A native Windows jq
# ends its lines with CRLF, which would leave a \r on every conclusion, so
# it is removed as from the checks file.
if ! runs="$(gh api --paginate "repos/$repo/commits/$sha/check-runs?filter=latest&per_page=100" |
  jq -r '.check_runs[] | [.name, .status, (.conclusion // "")] | @tsv' | tr -d '\r')"; then
  die "cannot list check runs for $repo@$sha"
fi

failed=0
total=0
while IFS= read -r check; do
  total=$((total + 1))
  found=0
  while IFS=$'\t' read -r name status conclusion; do
    [ "$name" = "$check" ] || continue
    found=1
    if [ "$status" != "completed" ]; then
      echo "::error::required check \"$check\" is $status on $sha"
      failed=1
    elif [ "$conclusion" != "success" ]; then
      echo "::error::required check \"$check\" concluded ${conclusion:-without a conclusion} on $sha"
      failed=1
    fi
  done <<< "$runs"
  if [ "$found" -eq 0 ]; then
    echo "::error::required check \"$check\" has no check run on $sha"
    failed=1
  fi
done <<< "$required"

if [ "$failed" -ne 0 ]; then
  die "$repo@$sha has not passed every check in $checks_file"
fi
echo "verify-ci: all $total required checks succeeded on $sha"
