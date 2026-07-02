#!/usr/bin/env bash
# install.sh — SupCode 一键安装脚本 (Linux / macOS)
set -euo pipefail

REPO="supcode/supcode"
BIN_DIR="${BIN_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

if [ "${BIN_DIR}" = "/usr/local/bin" ] && [ ! -w "${BIN_DIR}" ]; then
  echo "==> Need sudo to install to ${BIN_DIR}"
  SUDO="sudo"
else
  SUDO=""
fi

# ── Detect OS / Arch ──────────────────────────────────────
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "${OS}" in
  linux)   GOOS="linux" ;;
  darwin)  GOOS="darwin" ;;
  *)       echo "Unsupported OS: ${OS}"; exit 1 ;;
esac

case "${ARCH}" in
  x86_64|amd64) GOARCH="x86_64" ;;
  aarch64|arm64) GOARCH="arm64" ;;
  *)            echo "Unsupported arch: ${ARCH}"; exit 1 ;;
esac

# ── Resolve version ───────────────────────────────────────
if [ "${VERSION}" = "latest" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | cut -d'"' -f4)
  VERSION="${VERSION#v}"
fi

# ── Download ──────────────────────────────────────────────
ARCHIVE_NAME="supcode_${VERSION}_${GOOS}_${GOARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ARCHIVE_NAME}"
CHECKSUM_URL="${DOWNLOAD_URL}.sha256"

TMP_DIR=$(mktemp -d)
cd "${TMP_DIR}"

echo "==> Downloading ${DOWNLOAD_URL}"
curl -fsSL -o archive.tar.gz "${DOWNLOAD_URL}"

if command -v sha256sum &>/dev/null; then
  echo "==> Verifying checksum"
  curl -fsSL -o archive.tar.gz.sha256 "${CHECKSUM_URL}" 2>/dev/null || true
  if [ -f archive.tar.gz.sha256 ]; then
    sha256sum -c archive.tar.gz.sha256
  fi
fi

echo "==> Extracting"
tar xzf archive.tar.gz

# ── Install ───────────────────────────────────────────────
echo "==> Installing supcode to ${BIN_DIR}"
${SUDO} mv supcode "${BIN_DIR}/supcode"
${SUDO} chmod +x "${BIN_DIR}/supcode"

# ── Cleanup ──────────────────────────────────────────────
cd /
rm -rf "${TMP_DIR}"

echo "==> Installed $(supcode --version 2>/dev/null || echo 'supcode')"
echo "    Run 'supcode --help' to get started."
