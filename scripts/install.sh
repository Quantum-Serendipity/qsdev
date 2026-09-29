#!/bin/sh
# install.sh - qsdev installer for macOS and Linux
# Usage: curl -fsSL https://raw.githubusercontent.com/Quantum-Serendipity/qsdev/main/scripts/install.sh | sh
#    or: sh install.sh [--help] [--dry-run] [--no-modify-path] [--install-dir <dir>]
#
# Environment variables:
#   QSDEV_INSTALL_VERSION    Pin to a specific version (e.g. "1.2.3")
#   QSDEV_INSTALL_DIR        Override install directory (default: ~/.qsdev/bin)
#   QSDEV_ALLOW_UNSIGNED     "1": install even if the Sigstore bundle is missing
#   QSDEV_REQUIRE_SIGNATURE  "1": fail unless the Sigstore signature is verified
#   NO_COLOR                 Disable colored output
#
# QSDEV_VERSION is deliberately NOT a version pin: every qsdev-generated
# dev environment exports it (the version that generated the project), so
# honoring it would silently reinstall that version on upgrade.

set -eu

# --- Configuration ---
GITHUB_ORG="Quantum-Serendipity"
GITHUB_REPO="qsdev"
BINARY_NAME="qsdev"
DEFAULT_INSTALL_DIR="${HOME}/.qsdev/bin"
RECEIPT_NAME=".${BINARY_NAME}-version"

# --- Defaults for flags ---
DRY_RUN=false
NO_MODIFY_PATH=false
NO_VERIFY=false
VERIFY_ONLY=false
FORCE_ARCH=""
INSTALL_DIR="${QSDEV_INSTALL_DIR:-${DEFAULT_INSTALL_DIR}}"
VERSION="${QSDEV_INSTALL_VERSION:-}"
VERSION_SOURCE=""
if [ -n "${VERSION}" ]; then VERSION_SOURCE="QSDEV_INSTALL_VERSION"; fi
ALLOW_UNSIGNED=false
if [ "${QSDEV_ALLOW_UNSIGNED:-}" = "1" ]; then ALLOW_UNSIGNED=true; fi
REQUIRE_SIGNATURE=false
if [ "${QSDEV_REQUIRE_SIGNATURE:-}" = "1" ]; then REQUIRE_SIGNATURE=true; fi

# Temporary paths, removed by cleanup() on exit.
tmp_dir=""
tmp_bin=""

# --- Color support ---
setup_colors() {
    if [ -n "${NO_COLOR:-}" ] || [ "${TERM:-}" = "dumb" ] || ! [ -t 1 ]; then
        RED=""
        GREEN=""
        YELLOW=""
        CYAN=""
        BOLD=""
        RESET=""
    else
        RED='\033[0;31m'
        GREEN='\033[0;32m'
        YELLOW='\033[0;33m'
        CYAN='\033[0;36m'
        BOLD='\033[1m'
        RESET='\033[0m'
    fi
}

# --- Logging helpers ---
info() {
    printf "${CYAN}%s${RESET}\n" "$*"
}

success() {
    printf "${GREEN}%s${RESET}\n" "$*"
}

warn() {
    printf "${YELLOW}Warning: %s${RESET}\n" "$*" >&2
}

error() {
    printf "${RED}Error: %s${RESET}\n" "$*" >&2
}

# --- Usage ---
usage() {
    cat <<'USAGE'
qsdev installer

Usage:
  install.sh [options]

Options:
  --help              Show this help message
  --dry-run           Show what would be done without making changes
  --no-modify-path    Skip adding the install directory to shell PATH
  --install-dir DIR   Override the install directory (default: ~/.qsdev/bin)
  --version VERSION   Install (or verify) a specific version (e.g. 1.2.3)
  --verify-only       Verify an installed binary against the signed release
                      (exit 0 verified, 1 mismatch or error, 2 cannot verify
                      a package-manager install)
  --no-verify         Skip all verification (SHA256 and Sigstore)
  --allow-unsigned    Install even if the release has no Sigstore bundle
  --require-signature Fail unless the Sigstore signature is verified
                      (requires cosign)
  --force-arch ARCH   Override detected architecture (x86_64 or arm64)

Environment variables:
  QSDEV_INSTALL_VERSION    Pin to a specific version (e.g. "1.2.3")
  QSDEV_INSTALL_DIR        Override install directory
  QSDEV_ALLOW_UNSIGNED     Set to 1 for --allow-unsigned
  QSDEV_REQUIRE_SIGNATURE  Set to 1 for --require-signature
  NO_COLOR                 Disable colored output

Examples:
  # Install latest version
  curl -fsSL https://raw.githubusercontent.com/Quantum-Serendipity/qsdev/main/scripts/install.sh | sh

  # Install specific version
  QSDEV_INSTALL_VERSION=1.2.3 sh install.sh

  # Install to custom directory
  sh install.sh --install-dir /usr/local/bin

  # Preview without installing
  sh install.sh --dry-run

  # Verify an existing installation
  sh install.sh --verify-only
USAGE
}

cleanup() {
    if [ -n "${tmp_dir}" ]; then rm -rf "${tmp_dir}"; fi
    if [ -n "${tmp_bin}" ]; then rm -f "${tmp_bin}"; fi
}

# --- Version validation ---
# normalize_version strips a leading "v" and rejects anything that is not a
# release version, so a stray value (e.g. a git-describe string) fails with a
# clear message instead of a download 404.
normalize_version() {
    VERSION="${VERSION#v}"
    if ! printf '%s\n' "${VERSION}" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$'; then
        error "Invalid version '${VERSION}' (from ${VERSION_SOURCE:-unknown}); expected e.g. 1.2.3 or 1.2.3-rc.1."
        exit 1
    fi
}

# --- Platform detection ---
detect_platform() {
    OS_RAW="$(uname -s)"
    ARCH_RAW="$(uname -m)"

    case "${OS_RAW}" in
        Linux*)  OS="linux";  OS_TITLE="Linux" ;;
        Darwin*) OS="darwin"; OS_TITLE="Darwin" ;;
        MINGW*|MSYS*|CYGWIN*)
            error "Windows detected. Please use install.ps1 instead."
            exit 1
            ;;
        *)
            error "Unsupported operating system: ${OS_RAW}"
            exit 1
            ;;
    esac

    case "${ARCH_RAW}" in
        x86_64|amd64)  ARCH="x86_64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *)
            error "Unsupported architecture: ${ARCH_RAW}"
            exit 1
            ;;
    esac

    # On macOS, prefer native arm64 if running under Rosetta 2 translation.
    if [ "${OS}" = "darwin" ] && [ "${ARCH}" = "x86_64" ]; then
        if sysctl -n sysctl.proc_translated 2>/dev/null | grep -q "1"; then
            warn "Rosetta 2 detected; switching to native arm64 binary."
            ARCH="arm64"
        fi
    fi

    # Allow explicit architecture override.
    if [ -n "${FORCE_ARCH}" ]; then
        ARCH="${FORCE_ARCH}"
    fi
}

# --- musl libc detection (informational) ---
detect_musl() {
    if [ "${OS}" != "linux" ]; then return; fi
    if [ -f /etc/alpine-release ]; then
        info "Alpine Linux detected. Binary is statically linked; no compatibility issues."
    elif ldd --version 2>&1 | grep -qi musl 2>/dev/null; then
        info "musl libc detected. Binary is statically linked; no compatibility issues."
    fi
}

# --- Version resolution ---
resolve_version() {
    if [ -n "${VERSION}" ]; then
        return
    fi
    if [ -n "${QSDEV_VERSION:-}" ]; then
        info "Ignoring QSDEV_VERSION=${QSDEV_VERSION} (exported by qsdev-generated dev environments); set QSDEV_INSTALL_VERSION to pin a version."
    fi

    info "Fetching latest version..."
    VERSION="$(download "https://api.github.com/repos/${GITHUB_ORG}/${GITHUB_REPO}/releases/latest" - \
        | grep '"tag_name"' \
        | sed -E 's/.*"v([^"]+)".*/\1/')"

    if [ -z "${VERSION}" ]; then
        error "Could not determine latest version. Set QSDEV_INSTALL_VERSION or pass --version to install a specific version."
        exit 1
    fi
    VERSION_SOURCE="latest release"
}

# --- Download helper (curl/wget fallback) ---
download() {
    url="$1"
    output="$2"

    if command -v curl >/dev/null 2>&1; then
        if [ "${output}" = "-" ]; then
            curl -fsSL "${url}"
        else
            curl -fsSL "${url}" -o "${output}"
        fi
    elif command -v wget >/dev/null 2>&1; then
        if [ "${output}" = "-" ]; then
            wget -qO- "${url}"
        else
            wget -q "${url}" -O "${output}"
        fi
    else
        error "Either curl or wget is required to download files."
        exit 1
    fi
}

# --- SHA256 helper ---
sha256() {
    file="$1"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "${file}" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "${file}" | awk '{print $1}'
    else
        return 1
    fi
}

# --- Sigstore verification ---
# Every release publishes checksums.txt.sigstore.json, so once cosign is
# available a missing bundle is treated as tampering (an attacker who can
# replace the archive and checksums.txt can also delete the bundle) unless
# --allow-unsigned is given.
verify_sigstore() {
    if [ "${NO_VERIFY}" = true ]; then return; fi
    if [ "${DRY_RUN}" = true ]; then
        info "[dry-run] Would verify Sigstore signature on checksums.txt"
        return
    fi

    if ! command -v cosign >/dev/null 2>&1; then
        if [ "${REQUIRE_SIGNATURE}" = true ]; then
            error "cosign not found, but a verified Sigstore signature is required (--require-signature)."
            exit 1
        fi
        info "cosign not found; skipping Sigstore verification. Install cosign for enhanced security."
        return
    fi

    # cosign v1 has no --bundle and cannot check this release's signature.
    if ! cosign verify-blob --help 2>&1 | grep -q -- "--bundle"; then
        if [ "${REQUIRE_SIGNATURE}" = true ]; then
            error "Installed cosign does not support --bundle; upgrade cosign to verify the signature (--require-signature)."
            exit 1
        fi
        warn "cosign version does not support --bundle flag; skipping Sigstore verification."
        return
    fi

    bundle_url="https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/releases/download/v${VERSION}/checksums.txt.sigstore.json"
    bundle_path="${tmp_dir}/checksums.txt.sigstore.json"

    info "Verifying Sigstore signature on checksums.txt..."
    bundle_rc=0
    download "${bundle_url}" "${bundle_path}" 2>/dev/null || bundle_rc=$?
    if [ "${bundle_rc}" -ne 0 ]; then
        case "${bundle_rc}" in
            22|8) reason="the server returned an HTTP error (e.g. 404: not published)" ;;
            *)    reason="the download failed (exit ${bundle_rc}; network error?)" ;;
        esac
        if [ "${ALLOW_UNSIGNED}" = true ]; then
            warn "Sigstore bundle unavailable: ${reason}."
            warn "INSTALLING WITHOUT SIGNATURE VERIFICATION (--allow-unsigned). The checksum alone does not detect replaced release assets."
            return
        fi
        error "Sigstore bundle unavailable: ${reason}."
        error "Every qsdev release is signed; a missing bundle can mean the release assets were tampered with. Nothing was installed."
        error "Re-run with --allow-unsigned (or QSDEV_ALLOW_UNSIGNED=1) only if you have verified this release another way."
        exit 1
    fi

    expected_identity="https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/.github/workflows/release.yml@refs/tags/v${VERSION}"

    if cosign verify-blob \
        --bundle "${bundle_path}" \
        --certificate-identity "${expected_identity}" \
        --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
        "${tmp_dir}/checksums.txt"; then
        success "Sigstore signature verified."
    else
        error "Sigstore verification FAILED. The checksums file may have been tampered with."
        exit 1
    fi
}

# --- Download and verify ---
download_and_verify() {
    filename="${BINARY_NAME}_${VERSION}_${OS_TITLE}_${ARCH}.tar.gz"
    archive_url="https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/releases/download/v${VERSION}/${filename}"
    checksum_url="https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/releases/download/v${VERSION}/checksums.txt"

    info "Downloading ${BINARY_NAME} v${VERSION} for ${OS_TITLE}/${ARCH}..."

    if [ "${DRY_RUN}" = true ]; then
        info "[dry-run] Would download: ${archive_url}"
        info "[dry-run] Would download: ${checksum_url}"
        return
    fi

    download "${archive_url}" "${tmp_dir}/${filename}"
    download "${checksum_url}" "${tmp_dir}/checksums.txt"

    if [ "${NO_VERIFY}" = true ]; then
        warn "Verification skipped (--no-verify). The downloaded binary has not been verified."
        return
    fi

    info "Verifying SHA256 checksum..."
    expected_checksum="$(awk -v f="${filename}" '$2 == f {print $1}' "${tmp_dir}/checksums.txt")"

    if [ -z "${expected_checksum}" ]; then
        error "Could not find checksum for ${filename} in checksums.txt"
        exit 1
    fi

    actual_checksum="$(sha256 "${tmp_dir}/${filename}")" || {
        # Without the archive's checksum, a signature over checksums.txt
        # proves nothing about the archive.
        if [ "${REQUIRE_SIGNATURE}" = true ]; then
            error "Cannot verify checksum (neither sha256sum nor shasum available), so the signature cannot be checked (--require-signature)."
            exit 1
        fi
        warn "Cannot verify checksum (neither sha256sum nor shasum available). Proceeding without verification."
        return
    }

    if [ "${expected_checksum}" != "${actual_checksum}" ]; then
        error "Checksum verification failed!"
        error "  Expected: ${expected_checksum}"
        error "  Got:      ${actual_checksum}"
        exit 1
    fi

    success "Checksum verified."

    verify_sigstore
}

# --- Extract the downloaded archive into ${tmp_dir}/extracted ---
extract_archive() {
    mkdir -p "${tmp_dir}/extracted"
    tar -xzf "${tmp_dir}/${filename}" -C "${tmp_dir}/extracted"
}

# --- Extract and install ---
extract_and_install() {
    if [ "${DRY_RUN}" = true ]; then
        info "[dry-run] Would extract to: ${tmp_dir}/extracted/"
        info "[dry-run] Would install ${BINARY_NAME} to: ${INSTALL_DIR}/${BINARY_NAME}"
        return
    fi

    info "Installing to ${INSTALL_DIR}..."
    mkdir -p "${INSTALL_DIR}"
    extract_archive

    # Stage next to the target and rename over it: writing into a running
    # executable fails with "Text file busy" (qsdev's MCP server and hooks are
    # often running), and an interrupted copy must never leave a truncated
    # hook binary in place.
    tmp_bin="$(mktemp "${INSTALL_DIR}/.${BINARY_NAME}.new.XXXXXX")"
    cp "${tmp_dir}/extracted/${BINARY_NAME}" "${tmp_bin}"
    chmod 0755 "${tmp_bin}"
    mv -f "${tmp_bin}" "${INSTALL_DIR}/${BINARY_NAME}"
    tmp_bin=""
    # Install receipt: tells --verify-only which release to check against
    # without executing the binary under verification.
    printf '%s\n' "${VERSION}" > "${INSTALL_DIR}/${RECEIPT_NAME}"

    success "${BINARY_NAME} v${VERSION} installed to ${INSTALL_DIR}/${BINARY_NAME}"
}

# --- PATH setup ---
setup_path() {
    if [ "${NO_MODIFY_PATH}" = true ]; then
        return
    fi

    # Check if already on PATH
    case ":${PATH}:" in
        *":${INSTALL_DIR}:"*)
            return
            ;;
    esac

    # rc_file and path_line are set by detect_shell_rc()
    if [ -z "${rc_file}" ]; then
        warn "Unrecognized shell: $(basename "${SHELL:-/bin/sh}")"
        info "Manually add ${INSTALL_DIR} to your PATH."
        return
    fi

    if [ "${DRY_RUN}" = true ]; then
        info "[dry-run] Would append to ${rc_file}:"
        info "[dry-run]   ${path_line}"
        return
    fi

    if [ ! -f "${rc_file}" ]; then
        warn "Shell RC file not found: ${rc_file}"
        info "Manually add ${INSTALL_DIR} to your PATH:"
        info "  ${path_line}"
        return
    fi

    # Idempotent check: don't add if the install dir is already referenced
    if grep -qF "${INSTALL_DIR}" "${rc_file}" 2>/dev/null; then
        return
    fi

    printf '\n# Added by qsdev installer\n%s\n' "${path_line}" >> "${rc_file}"
    success "Added ${INSTALL_DIR} to PATH in ${rc_file}"
}

# --- Verify existing installation ---
# Downloads the release archive for the installed version, verifies it like an
# install (checksum and Sigstore, fail closed), and compares the binary inside
# with the installed one. The installed binary is never executed: it is the
# file under suspicion. Exit codes: 0 verified, 1 mismatch or error, 2 cannot
# verify (a package-manager install, which is not a release archive copy).
run_verify_only() {
    installed_path=""
    if [ -x "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        installed_path="${INSTALL_DIR}/${BINARY_NAME}"
    elif command -v "${BINARY_NAME}" >/dev/null 2>&1; then
        installed_path="$(command -v "${BINARY_NAME}")"
    fi

    if [ -z "${installed_path}" ]; then
        error "${BINARY_NAME} is not installed. Nothing to verify."
        exit 1
    fi

    resolved_path="$(readlink -f "${installed_path}" 2>/dev/null || true)"
    case "${resolved_path:-${installed_path}}" in
        /nix/store/*|*/Cellar/*|/opt/homebrew/*|/home/linuxbrew/*)
            warn "${installed_path} is managed by a package manager (${resolved_path}); it is not a copy of a release archive and cannot be verified here."
            warn "Use the package manager's own integrity checks instead."
            exit 2
            ;;
    esac

    receipt="$(dirname "${installed_path}")/${RECEIPT_NAME}"
    if [ -z "${VERSION}" ] && [ -f "${receipt}" ]; then
        VERSION="$(head -n 1 "${receipt}")"
        VERSION_SOURCE="install receipt ${receipt}"
    fi
    if [ -z "${VERSION}" ]; then
        error "Cannot tell which release ${installed_path} came from (no install receipt at ${receipt})."
        error "Re-run with --version <version>; the binary is not run to ask, since it is the file being verified."
        exit 1
    fi
    normalize_version

    info "Verifying ${installed_path} against release v${VERSION} (version from ${VERSION_SOURCE})..."
    if [ "${DRY_RUN}" = true ]; then
        info "[dry-run] Would download, verify and compare the v${VERSION} release binary"
        return
    fi

    tmp_dir="$(mktemp -d)"
    download_and_verify
    extract_archive

    if ! expected_checksum="$(sha256 "${tmp_dir}/extracted/${BINARY_NAME}")" ||
        ! actual_checksum="$(sha256 "${installed_path}")"; then
        error "Cannot compute checksum (neither sha256sum nor shasum available)."
        exit 1
    fi

    if [ "${expected_checksum}" != "${actual_checksum}" ]; then
        error "${installed_path} does NOT match the qsdev v${VERSION} release binary."
        error "  Expected (release binary): ${expected_checksum}"
        error "  Got (installed):           ${actual_checksum}"
        exit 1
    fi
    success "${installed_path} matches the signed qsdev v${VERSION} release binary."
}

# --- Parse arguments ---
parse_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            --help|-h)
                usage
                exit 0
                ;;
            --dry-run)
                DRY_RUN=true
                ;;
            --no-modify-path)
                NO_MODIFY_PATH=true
                ;;
            --no-verify)
                NO_VERIFY=true
                ;;
            --verify-only)
                VERIFY_ONLY=true
                ;;
            --allow-unsigned)
                ALLOW_UNSIGNED=true
                ;;
            --require-signature)
                REQUIRE_SIGNATURE=true
                ;;
            --version)
                if [ $# -lt 2 ]; then
                    error "--version requires an argument (e.g. 1.2.3)"
                    exit 1
                fi
                VERSION="$2"
                VERSION_SOURCE="--version"
                shift
                ;;
            --force-arch)
                if [ $# -lt 2 ]; then
                    error "--force-arch requires an argument (x86_64 or arm64)"
                    exit 1
                fi
                case "$2" in
                    x86_64|arm64) FORCE_ARCH="$2" ;;
                    *)
                        error "Invalid architecture: $2 (must be x86_64 or arm64)"
                        exit 1
                        ;;
                esac
                shift
                ;;
            --install-dir)
                if [ $# -lt 2 ]; then
                    error "--install-dir requires an argument"
                    exit 1
                fi
                INSTALL_DIR="$2"
                shift
                ;;
            *)
                error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
        shift
    done
}

# --- Determine shell RC file ---
detect_shell_rc() {
    current_shell="$(basename "${SHELL:-/bin/sh}")"
    rc_file=""
    path_line=""

    case "${current_shell}" in
        bash)
            rc_file="${HOME}/.bashrc"
            path_line="export PATH=\"${INSTALL_DIR}:\$PATH\""
            ;;
        zsh)
            rc_file="${HOME}/.zshrc"
            path_line="export PATH=\"${INSTALL_DIR}:\$PATH\""
            ;;
        fish)
            rc_file="${HOME}/.config/fish/config.fish"
            path_line="set -gx PATH \"${INSTALL_DIR}\" \$PATH"
            ;;
    esac
}

# --- Main ---
main() {
    setup_colors
    parse_args "$@"

    trap cleanup EXIT
    # An interrupted run exits through the EXIT trap, so staging files are
    # removed (some shells skip EXIT traps on a fatal signal).
    trap 'exit 130' INT
    trap 'exit 143' TERM
    trap 'exit 129' HUP

    if [ "${ALLOW_UNSIGNED}" = true ] && [ "${REQUIRE_SIGNATURE}" = true ]; then
        error "--allow-unsigned and --require-signature (or QSDEV_ALLOW_UNSIGNED and QSDEV_REQUIRE_SIGNATURE) contradict each other."
        exit 1
    fi

    detect_platform
    detect_musl

    if [ "${VERIFY_ONLY}" = true ]; then
        if [ "${NO_VERIFY}" = true ]; then
            error "--verify-only cannot be combined with --no-verify."
            exit 1
        fi
        run_verify_only
        exit 0
    fi

    resolve_version
    normalize_version
    detect_shell_rc

    printf '%sqsdev installer%s\n' "${BOLD}" "${RESET}"
    printf "%s\n" "==============="
    info "  Version:     v${VERSION} (${VERSION_SOURCE})"
    info "  OS:          ${OS_TITLE}"
    info "  Arch:        ${ARCH}"
    info "  Install dir: ${INSTALL_DIR}"
    printf "\n"

    if [ "${DRY_RUN}" = true ]; then
        info "[dry-run mode -- no changes will be made]"
        printf "\n"
    fi

    tmp_dir="$(mktemp -d)"

    download_and_verify
    extract_and_install
    setup_path

    printf "\n"
    success "Installation complete!"
    printf "\n"
    info "Next steps:"
    if [ -n "${rc_file}" ]; then
        info "  1. Restart your shell or run:  source ${rc_file}"
    else
        info "  1. Restart your shell"
    fi
    info "  2. Verify the installation:    ${BINARY_NAME} version"
    info "  3. Check your environment:     ${BINARY_NAME} doctor"
    info "  4. Initialize a project:       ${BINARY_NAME} init"
}

main "$@"
