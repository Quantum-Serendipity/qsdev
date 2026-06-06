#!/bin/sh
# install.sh - qsdev installer for macOS and Linux
# Usage: curl -fsSL https://raw.githubusercontent.com/Quantum-Serendipity/qsdev/main/scripts/install.sh | sh
#    or: sh install.sh [--help] [--dry-run] [--no-modify-path] [--install-dir <dir>]
#
# Environment variables:
#   QSDEV_VERSION      Pin to a specific version (e.g. "1.2.3")
#   QSDEV_INSTALL_DIR  Override install directory (default: ~/.qsdev/bin)
#   NO_COLOR            Disable colored output

set -eu

# --- Configuration ---
GITHUB_ORG="Quantum-Serendipity"
GITHUB_REPO="qsdev"
BINARY_NAME="qsdev"
DEFAULT_INSTALL_DIR="${HOME}/.qsdev/bin"

# --- Defaults for flags ---
DRY_RUN=false
NO_MODIFY_PATH=false
NO_VERIFY=false
VERIFY_ONLY=false
FORCE_ARCH=""
INSTALL_DIR="${QSDEV_INSTALL_DIR:-${DEFAULT_INSTALL_DIR}}"
VERSION="${QSDEV_VERSION:-}"

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
  --verify-only       Verify an existing installation against release checksums
  --no-verify         Skip all verification (SHA256 and Sigstore)
  --force-arch ARCH   Override detected architecture (x86_64 or arm64)

Environment variables:
  QSDEV_VERSION       Pin to a specific version (e.g. "1.2.3")
  QSDEV_INSTALL_DIR   Override install directory
  NO_COLOR            Disable colored output

Examples:
  # Install latest version
  curl -fsSL https://raw.githubusercontent.com/Quantum-Serendipity/qsdev/main/scripts/install.sh | sh

  # Install specific version
  QSDEV_VERSION=1.2.3 sh install.sh

  # Install to custom directory
  sh install.sh --install-dir /usr/local/bin

  # Preview without installing
  sh install.sh --dry-run

  # Verify an existing installation
  sh install.sh --verify-only
USAGE
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

    info "Fetching latest version..."
    VERSION="$(download "https://api.github.com/repos/${GITHUB_ORG}/${GITHUB_REPO}/releases/latest" - \
        | grep '"tag_name"' \
        | sed -E 's/.*"v([^"]+)".*/\1/')"

    if [ -z "${VERSION}" ]; then
        error "Could not determine latest version. Set QSDEV_VERSION to install a specific version."
        exit 1
    fi
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
verify_sigstore() {
    if [ "${NO_VERIFY}" = true ]; then return; fi
    if [ "${DRY_RUN}" = true ]; then
        info "[dry-run] Would verify Sigstore signature on checksums.txt"
        return
    fi

    if ! command -v cosign >/dev/null 2>&1; then
        info "cosign not found; skipping Sigstore verification. Install cosign for enhanced security."
        return
    fi

    bundle_url="https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/releases/download/v${VERSION}/checksums.txt.sigstore.json"
    bundle_path="${tmp_dir}/checksums.txt.sigstore.json"

    info "Verifying Sigstore signature on checksums.txt..."
    if ! download "${bundle_url}" "${bundle_path}" 2>/dev/null; then
        info "Sigstore bundle not found for this release; skipping signature verification."
        return
    fi

    expected_identity="https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/.github/workflows/release.yml@refs/tags/v${VERSION}"

    if cosign verify-blob \
        --bundle "${bundle_path}" \
        --certificate-identity "${expected_identity}" \
        --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
        "${tmp_dir}/checksums.txt" 2>/dev/null; then
        success "Sigstore signature verified."
    else
        # Check if cosign doesn't support --bundle (v1). Treat as unusable.
        if cosign verify-blob --help 2>&1 | grep -q "\-\-bundle"; then
            error "Sigstore verification FAILED. The checksums file may have been tampered with."
            exit 1
        else
            warn "cosign version does not support --bundle flag; skipping Sigstore verification."
        fi
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

# --- Extract and install ---
extract_and_install() {
    if [ "${DRY_RUN}" = true ]; then
        info "[dry-run] Would extract to: ${tmp_dir}/extracted/"
        info "[dry-run] Would install ${BINARY_NAME} to: ${INSTALL_DIR}/${BINARY_NAME}"
        return
    fi

    info "Installing to ${INSTALL_DIR}..."
    mkdir -p "${INSTALL_DIR}"
    mkdir -p "${tmp_dir}/extracted"
    tar -xzf "${tmp_dir}/${filename}" -C "${tmp_dir}/extracted"

    cp "${tmp_dir}/extracted/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

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
run_verify_only() {
    # Locate installed binary.
    installed_path=""
    if command -v "${BINARY_NAME}" >/dev/null 2>&1; then
        installed_path="$(command -v "${BINARY_NAME}")"
    elif [ -x "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        installed_path="${INSTALL_DIR}/${BINARY_NAME}"
    fi

    if [ -z "${installed_path}" ]; then
        error "${BINARY_NAME} is not installed. Nothing to verify."
        exit 1
    fi

    # Get installed version (output format: "qsdev version 0.7.3+abc123").
    installed_version="$("${installed_path}" version 2>/dev/null | head -1 | awk '{print $3}')" || true
    # Strip build metadata (e.g. "+abc123") and v prefix.
    installed_version="$(echo "${installed_version}" | sed 's/+.*//' | sed 's/^v//')"

    if [ -z "${installed_version}" ] || [ "${installed_version}" = "dev" ]; then
        error "Cannot determine installed version (got: ${installed_version:-empty})."
        exit 1
    fi

    VERSION="${installed_version}"
    info "Verifying ${BINARY_NAME} v${VERSION} at ${installed_path}..."

    # Set up temp directory with cleanup trap.
    tmp_dir="$(mktemp -d)"
    trap 'rm -rf "$tmp_dir"' EXIT

    # Download checksums and verify.
    filename="${BINARY_NAME}_${VERSION}_${OS_TITLE}_${ARCH}.tar.gz"
    checksum_url="https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/releases/download/v${VERSION}/checksums.txt"

    download "${checksum_url}" "${tmp_dir}/checksums.txt"

    expected_checksum="$(awk -v f="${filename}" '$2 == f {print $1}' "${tmp_dir}/checksums.txt")"
    if [ -z "${expected_checksum}" ]; then
        warn "No checksum found for ${filename} — cannot verify binary directly."
        warn "The installed binary may have been built from source or installed via a package manager."
        exit 0
    fi

    actual_checksum="$(sha256 "${installed_path}")" || {
        error "Cannot compute checksum (neither sha256sum nor shasum available)."
        exit 1
    }

    if [ "${expected_checksum}" = "${actual_checksum}" ]; then
        success "SHA256 checksum matches release artifact."
    else
        warn "SHA256 checksum does NOT match the release archive."
        warn "This is expected if the binary was installed via a package manager (Homebrew, Nix, AUR)."
        warn "  Expected (archive): ${expected_checksum}"
        warn "  Got (binary):       ${actual_checksum}"
    fi

    verify_sigstore

    success "Verification complete."
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

    detect_platform
    detect_musl

    if [ "${VERIFY_ONLY}" = true ]; then
        resolve_version 2>/dev/null || true
        run_verify_only
        exit 0
    fi

    resolve_version
    detect_shell_rc

    printf '%sqsdev installer%s\n' "${BOLD}" "${RESET}"
    printf "%s\n" "==============="
    info "  Version:     v${VERSION}"
    info "  OS:          ${OS_TITLE}"
    info "  Arch:        ${ARCH}"
    info "  Install dir: ${INSTALL_DIR}"
    printf "\n"

    if [ "${DRY_RUN}" = true ]; then
        info "[dry-run mode -- no changes will be made]"
        printf "\n"
    fi

    # Set up temp directory with cleanup trap
    tmp_dir="$(mktemp -d)"
    trap 'rm -rf "$tmp_dir"' EXIT

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
