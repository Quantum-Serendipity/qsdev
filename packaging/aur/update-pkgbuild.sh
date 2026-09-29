#!/bin/sh
# update-pkgbuild.sh — Update PKGBUILD and .SRCINFO with the version and
# checksums of a published release.
# Usage: ./update-pkgbuild.sh VERSION
#   VERSION must be a published release (e.g. 0.7.9). It is not defaulted from
#   ../../VERSION, which names the NEXT release and has no assets yet.
#
# The checksums come from the release's checksums.txt. When cosign is
# installed, that file's Sigstore signature is verified first and a missing or
# invalid signature aborts the update.

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "${SCRIPT_DIR}"

if [ $# -ne 1 ]; then
    echo "Usage: $0 VERSION" >&2
    exit 1
fi
VERSION="${1#v}"
REPO="https://github.com/Quantum-Serendipity/qsdev"
BASE_URL="${REPO}/releases/download/v${VERSION}"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

echo "Updating PKGBUILD for v${VERSION}..."

curl -fsSL -o "${tmp_dir}/checksums.txt" "${BASE_URL}/checksums.txt"

if command -v cosign >/dev/null 2>&1; then
    curl -fsSL -o "${tmp_dir}/checksums.txt.sigstore.json" "${BASE_URL}/checksums.txt.sigstore.json"
    cosign verify-blob \
        --bundle "${tmp_dir}/checksums.txt.sigstore.json" \
        --certificate-identity "${REPO}/.github/workflows/release.yml@refs/tags/v${VERSION}" \
        --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
        "${tmp_dir}/checksums.txt"
else
    echo "  Warning: cosign not found; checksums.txt signature not verified." >&2
fi

checksum_of() {
    awk -v f="$1" '$2 == f {print $1}' "${tmp_dir}/checksums.txt"
}
hash_x86="$(checksum_of "qsdev_${VERSION}_Linux_x86_64.tar.gz")"
hash_arm="$(checksum_of "qsdev_${VERSION}_Linux_arm64.tar.gz")"

for h in "${hash_x86}" "${hash_arm}"; do
    if ! printf '%s\n' "${h}" | grep -Eq '^[0-9a-f]{64}$'; then
        echo "Error: missing or malformed Linux archive checksum in checksums.txt" >&2
        exit 1
    fi
done

sed -i "s/^pkgver=.*/pkgver=${VERSION}/" PKGBUILD
sed -i "s/^pkgrel=.*/pkgrel=1/" PKGBUILD
sed -i "s/^sha256sums_x86_64=.*/sha256sums_x86_64=('${hash_x86}')/" PKGBUILD
sed -i "s/^sha256sums_aarch64=.*/sha256sums_aarch64=('${hash_arm}')/" PKGBUILD

if command -v makepkg >/dev/null 2>&1; then
    makepkg --printsrcinfo > .SRCINFO
else
    # Keep .SRCINFO in step without makepkg: it mirrors these PKGBUILD fields.
    sed -i \
        -e "s/^\(\tpkgver = \).*/\1${VERSION}/" \
        -e "s/^\(\tpkgrel = \).*/\11/" \
        -e "s#/releases/download/v[^/]*/qsdev_[^_]*_#/releases/download/v${VERSION}/qsdev_${VERSION}_#" \
        -e "s/^\(\tsha256sums_x86_64 = \).*/\1${hash_x86}/" \
        -e "s/^\(\tsha256sums_aarch64 = \).*/\1${hash_arm}/" \
        .SRCINFO
fi

echo "  pkgver=${VERSION}"
echo "  sha256sums_x86_64=${hash_x86}"
echo "  sha256sums_aarch64=${hash_arm}"
echo "Done."
