#!/usr/bin/env bash
# install-nix.sh: install the pinned official Nix release on a GitHub Actions
# runner, with no third-party action. The repository's Actions allowlist does
# not admit cachix/install-nix-action (TestWorkflowActionsAllowlisted).
#
# Usage: .github/scripts/install-nix.sh [ENV_FILE]
#
# NIX_VERSION and the per-system NIX_SHA256_* digests come from ENV_FILE
# (default .github/tool-versions.env, the single source of tool pins, kept
# current by tool-pins.yml). The file is parsed, never sourced.
#
# The script downloads nix-<version>-<system>.tar.xz from releases.nixos.org
# and checks it against the pinned SHA-256 before anything is extracted or run.
# It then runs the tarball's bundled installer: multi-user (--daemon) on macOS,
# the only mode Nix supports there, and single-user (--no-daemon) on Linux. It
# enables nix-command and flakes, which the workflows' nix build, eval, flake,
# shell and registry commands need, and puts nix on PATH for later steps
# through $GITHUB_PATH and $GITHUB_ENV.
#
# When GITHUB_TOKEN is set, it is written to the runner user's nix.conf (mode
# 0600) as a github.com access token, so flake fetches from GitHub are not
# rate limited as anonymous requests.
set -euo pipefail

readonly RELEASES_URL="https://releases.nixos.org/nix"

env_file="${1:-.github/tool-versions.env}"

die() {
  echo "install-nix: $*" >&2
  exit 1
}

[ -f "$env_file" ] || die "$env_file not found"
[ -n "${GITHUB_PATH:-}" ] || die "GITHUB_PATH is not set; run this in a GitHub Actions step"
[ -n "${GITHUB_ENV:-}" ] || die "GITHUB_ENV is not set; run this in a GitHub Actions step"

# pin KEY prints KEY's value from env_file, which must define it exactly once.
pin() {
  local line value="" count=0
  while IFS= read -r line || [ -n "$line" ]; do
    if [[ "$line" == "$1="* ]]; then
      value="${line#"$1="}"
      count=$((count + 1))
    fi
  done <"$env_file"
  [ "$count" -eq 1 ] || die "$1 is defined $count times in $env_file, want exactly once"
  printf '%s\n' "$value"
}

os="$(uname -s)"
case "$os-$(uname -m)" in
  Linux-x86_64) system=x86_64-linux key=NIX_SHA256_X86_64_LINUX ;;
  Linux-aarch64 | Linux-arm64) system=aarch64-linux key=NIX_SHA256_AARCH64_LINUX ;;
  Darwin-arm64) system=aarch64-darwin key=NIX_SHA256_AARCH64_DARWIN ;;
  Darwin-x86_64) system=x86_64-darwin key=NIX_SHA256_X86_64_DARWIN ;;
  *) die "no pinned Nix release for $os $(uname -m)" ;;
esac

version="$(pin NIX_VERSION)"
want="$(pin "$key")"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "NIX_VERSION=$version is not an exact release (want X.Y.Z)"
[[ "$want" =~ ^[0-9a-f]{64}$ ]] || die "$key=$want is not a SHA-256 hex digest"

work="$(mktemp -d "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/install-nix.XXXXXX")"
trap 'rm -rf "$work"' EXIT

tarball="nix-${version}-${system}.tar.xz"
echo "Downloading $RELEASES_URL/nix-${version}/${tarball}"
curl -fsSL --proto '=https' --tlsv1.2 --retry 3 -o "$work/$tarball" "$RELEASES_URL/nix-${version}/${tarball}"

# Verify before extracting: nothing from the tarball runs unless it is
# byte-for-byte the release that was pinned.
printf '%s  %s\n' "$want" "$tarball" >"$work/$tarball.sha256"
if command -v sha256sum >/dev/null 2>&1; then
  (cd "$work" && sha256sum --check --strict "$tarball.sha256")
else
  (cd "$work" && shasum -a 256 -c "$tarball.sha256")
fi

tar -xJf "$work/$tarball" -C "$work"
installer="$work/nix-${version}-${system}/install"
[ -x "$installer" ] || die "$tarball has no installer at nix-${version}-${system}/install"

# Settings every nix invocation in the workflows needs. The token, a secret,
# goes only into the user's own mode-0600 nix.conf below.
conf="$work/nix.conf"
cat >"$conf" <<'EOF'
experimental-features = nix-command flakes
max-jobs = auto
EOF

user_conf_dir="${XDG_CONFIG_HOME:-$HOME/.config}/nix"
mkdir -p "$user_conf_dir"
(
  umask 077
  if [ "$os" = Linux ]; then
    # A single-user install reads no /etc/nix/nix.conf; the user's file
    # carries every setting.
    cat "$conf" >>"$user_conf_dir/nix.conf"
  fi
  if [ -n "${GITHUB_TOKEN:-}" ]; then
    printf 'access-tokens = github.com=%s\n' "$GITHUB_TOKEN" >>"$user_conf_dir/nix.conf"
  fi
  [ ! -e "$user_conf_dir/nix.conf" ] || chmod 600 "$user_conf_dir/nix.conf"
)

if [ "$os" = Darwin ]; then
  "$installer" --daemon --yes --no-channel-add --nix-extra-conf-file "$conf"
  profile=/nix/var/nix/profiles/default
  echo "$profile/bin" >>"$GITHUB_PATH"
else
  # Ubuntu 24.04 confines unprivileged user namespaces with AppArmor, and a
  # single-user Nix builds its sandbox inside one.
  userns=/proc/sys/kernel/apparmor_restrict_unprivileged_userns
  if [ -r "$userns" ] && [ "$(cat "$userns")" != 0 ]; then
    sudo sysctl -w kernel.apparmor_restrict_unprivileged_userns=0
  fi
  "$installer" --no-daemon --yes --no-channel-add --no-modify-profile
  profile="$HOME/.nix-profile"
fi
# The user profile last, so it takes precedence as in Nix's own profile script.
echo "$HOME/.nix-profile/bin" >>"$GITHUB_PATH"

# Nix's profile script would point NIX_SSL_CERT_FILE at the system bundle when
# there is one and at the bundle the installer placed in the profile otherwise.
if [ -z "${NIX_SSL_CERT_FILE:-}" ]; then
  for cert in /etc/ssl/certs/ca-certificates.crt "$profile/etc/ssl/certs/ca-bundle.crt"; do
    if [ -f "$cert" ]; then
      echo "NIX_SSL_CERT_FILE=$cert" >>"$GITHUB_ENV"
      break
    fi
  done
fi

got="$("$profile/bin/nix" --version)"
[ "$got" = "nix (Nix) $version" ] || die "installed nix reports \"$got\", want \"nix (Nix) $version\""
echo "Installed $got for $system"
