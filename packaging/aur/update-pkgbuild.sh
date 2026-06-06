#!/bin/sh
# update-pkgbuild.sh — Update PKGBUILD with version and checksums from a release.
# Usage: ./update-pkgbuild.sh [VERSION]
#   VERSION defaults to the contents of ../../VERSION.

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "${SCRIPT_DIR}"

VERSION="${1:-$(cat ../../VERSION)}"
REPO="https://github.com/Quantum-Serendipity/qsdev"
CHECKSUMS_URL="${REPO}/releases/download/v${VERSION}/checksums.txt"

echo "Updating PKGBUILD for v${VERSION}..."

checksums="$(curl -fsSL "${CHECKSUMS_URL}")"

hash_x86="$(echo "${checksums}" | awk '/qsdev_.*_Linux_x86_64\.tar\.gz$/ {print $1}')"
hash_arm="$(echo "${checksums}" | awk '/qsdev_.*_Linux_arm64\.tar\.gz$/ {print $1}')"

if [ -z "${hash_x86}" ]; then
    echo "Error: could not find x86_64 checksum in checksums.txt" >&2
    exit 1
fi

if [ -z "${hash_arm}" ]; then
    echo "Error: could not find aarch64 checksum in checksums.txt" >&2
    exit 1
fi

sed -i "s/^pkgver=.*/pkgver=${VERSION}/" PKGBUILD
sed -i "s/^pkgrel=.*/pkgrel=1/" PKGBUILD
sed -i "s/^sha256sums_x86_64=.*/sha256sums_x86_64=('${hash_x86}')/" PKGBUILD
sed -i "s/^sha256sums_aarch64=.*/sha256sums_aarch64=('${hash_arm}')/" PKGBUILD

echo "  pkgver=${VERSION}"
echo "  sha256sums_x86_64=${hash_x86}"
echo "  sha256sums_aarch64=${hash_arm}"

if command -v makepkg >/dev/null 2>&1; then
    makepkg --printsrcinfo > .SRCINFO
    echo "  .SRCINFO regenerated."
else
    echo "  Warning: makepkg not found; .SRCINFO not regenerated."
fi

echo "Done."
