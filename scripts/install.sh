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

    info "Verifying SHA256 checksum..."
    expected_checksum="$(grep "${filename}" "${tmp_dir}/checksums.txt" | awk '{print $1}')"

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
    resolve_version
    detect_shell_rc

    printf "${BOLD}qsdev installer${RESET}\n"
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
