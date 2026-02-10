#!/bin/bash
#
# md2wechat - Minimal binary provisioner
# Philosophy: Simple, fast, single responsibility
#

set -e

# =============================================================================
# CONFIGURATION
# =============================================================================

VERSION="1.10.0"
REPO="geekjourneyx/md2wechat-skill"
BINARY_NAME="md2wechat"

# Cache directory (tool-specific, not Claude's cache)
CACHE_DIR="${XDG_CACHE_HOME:-${HOME}/.cache}/md2wechat"
BIN_DIR="${CACHE_DIR}/bin"
VERSION_FILE="${CACHE_DIR}/.version"

# Minimum binary size (100KB - protects against corrupted downloads)
MIN_BINARY_SIZE=102400

# =============================================================================
# PLATFORM DETECTION
# =============================================================================

detect_platform() {
    local os arch

    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)

    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) echo "Unsupported architecture: $arch" >&2; return 1 ;;
    esac

    case "$os" in
        darwin|linux) echo "${os}-${arch}" ;;
        msys*|mingw*|cygwin*) echo "windows-${arch}" ;;
        *) echo "Unsupported OS: $os" >&2; return 1 ;;
    esac
}

# =============================================================================
# BINARY MANAGEMENT
# =============================================================================

get_binary_path() {
    local platform=$1
    local path="${BIN_DIR}/${BINARY_NAME}-${platform}"
    [[ "$platform" == windows-* ]] && path="${path}.exe"
    echo "$path"
}

is_cache_valid() {
    local binary=$1
    [[ -x "$binary" ]] && [[ -f "$VERSION_FILE" ]] && [[ "$(cat "$VERSION_FILE" 2>/dev/null)" == "$VERSION" ]]
}

download_binary() {
    local platform=$1
    local binary=$2
    local bin_name="${BINARY_NAME}-${platform}"
    [[ "$platform" == windows-* ]] && bin_name="${bin_name}.exe"

    local url="https://github.com/${REPO}/releases/download/v${VERSION}/${bin_name}"
    local temp_file="${binary}.tmp"

    echo "  Downloading md2wechat v${VERSION} for ${platform}..." >&2

    # Check for download tool
    if command -v curl &>/dev/null; then
        curl -fsSL --connect-timeout 15 --max-time 120 -o "$temp_file" "$url" 2>/dev/null
    elif command -v wget &>/dev/null; then
        wget -q --timeout=120 -O "$temp_file" "$url" 2>/dev/null
    else
        echo "  Error: curl or wget required" >&2
        echo "  Install: brew install curl (macOS) or apt install curl (Linux)" >&2
        return 1
    fi

    # Validate file size
    local size
    size=$(wc -c < "$temp_file" 2>/dev/null | tr -d ' ') || size=0
    if [[ $size -lt $MIN_BINARY_SIZE ]]; then
        rm -f "$temp_file"
        echo "  Error: Download incomplete or corrupted" >&2
        return 1
    fi

    mv "$temp_file" "$binary"
    chmod +x "$binary"
    echo "$VERSION" > "$VERSION_FILE"
    echo "  Ready!" >&2
}

ensure_binary() {
    local platform
    platform=$(detect_platform) || return 1

    local binary
    binary=$(get_binary_path "$platform")

    # Fast path: valid cache
    if is_cache_valid "$binary"; then
        echo "$binary"
        return 0
    fi

    # Try local development binary
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local local_binary="${script_dir}/bin/${BINARY_NAME}-${platform}"
    [[ "$platform" == windows-* ]] && local_binary="${local_binary}.exe"

    if [[ -x "$local_binary" ]]; then
        echo "$local_binary"
        return 0
    fi

    # Download
    mkdir -p "$BIN_DIR" 2>/dev/null || {
        echo "  Error: Cannot create cache directory: $BIN_DIR" >&2
        return 1
    }

    if download_binary "$platform" "$binary"; then
        echo "$binary"
        return 0
    fi

    echo "" >&2
    echo "  Download failed. Try:" >&2
    echo "    • Check your network connection" >&2
    echo "    • Download manually: https://github.com/${REPO}/releases" >&2
    echo "" >&2
    return 1
}

# =============================================================================
# MAIN
# =============================================================================

main() {
    local binary
    binary=$(ensure_binary) || exit 1
    exec "$binary" "$@"
}

main "$@"
