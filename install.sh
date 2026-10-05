#!/usr/bin/env bash
set -e

# dots installer script
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/BitwiseSang/dots/main/install.sh | bash

REPO="BitwiseSang/dots"
BINARY="dots"

# 1. Detect OS
OS="$(uname -s)"
case "${OS}" in
    Linux*)     PLATFORM="Linux" ;;
    Darwin*)    PLATFORM="Darwin" ;;
    *)          echo "Error: Unsupported operating system: ${OS}"; exit 1 ;;
esac

# 2. Detect Architecture
ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64|amd64)    ARCH_NAME="amd64" ;;
    arm64|aarch64)   ARCH_NAME="arm64" ;;
    *)               echo "Error: Unsupported architecture: ${ARCH}"; exit 1 ;;
esac

# 3. Determine target install directory
if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "${INSTALL_DIR}"
fi

# 4. Fetch latest release version from GitHub API
echo "==> Fetching latest release information for ${REPO}..."
RELEASE_JSON="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || true)"

TAG=""
if [ -n "${RELEASE_JSON}" ]; then
    TAG="$(echo "${RELEASE_JSON}" | grep '"tag_name":' | head -n 1 | cut -d '"' -f 4)"
fi

if [ -z "${TAG}" ]; then
    TAG="v1.0.0"
fi

VERSION="${TAG#v}"
TARBALL="${BINARY}_${VERSION}_${PLATFORM}_${ARCH_NAME}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${TARBALL}"

echo "==> Downloading ${BINARY} ${TAG} for ${PLATFORM} (${ARCH_NAME})..."
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

if curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/${TARBALL}"; then
    tar -xzf "${TMP_DIR}/${TARBALL}" -C "${TMP_DIR}"
    install -m 755 "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    echo ""
    echo "✓ Successfully installed ${BINARY} to ${INSTALL_DIR}/${BINARY}!"
    
    # Check PATH
    case ":${PATH}:" in
        *":${INSTALL_DIR}:"*) ;;
        *)
            echo ""
            echo "⚠️  Note: ${INSTALL_DIR} is not in your PATH."
            echo "   Add it to your shell configuration:"
            echo "   export PATH=\"${INSTALL_DIR}:\$PATH\""
            ;;
    esac
    
    echo ""
    "${INSTALL_DIR}/${BINARY}" --version || true
else
    echo ""
    echo "Could not download pre-built binary from ${DOWNLOAD_URL}."
    echo "If a release is not yet published, you can install via Go:"
    echo "  go install github.com/${REPO}/cmd/dots@latest"
    exit 1
fi
