#!/bin/bash
set -e

: "${CLAUDE_PLUGIN_ROOT:?CLAUDE_PLUGIN_ROOT is not set}"

BIN_DIR="${CLAUDE_PLUGIN_ROOT}/bin"

# Detect architecture
ARCH=$(uname -m)
case "${ARCH}" in
  x86_64|amd64) BINARY="auto-gdb-linux-amd64" ;;
  aarch64|arm64) BINARY="auto-gdb-linux-arm64" ;;
  *) echo "Unsupported architecture: ${ARCH}" >&2; exit 1 ;;
esac

# Ensure target binary is executable (in case of fresh git clone)
chmod +x "${BIN_DIR}/${BINARY}" 2>/dev/null || true

# Create symlink to platform-specific binary
ln -sf "${BINARY}" "${BIN_DIR}/auto-gdb"

echo "Installed auto-gdb (${BINARY}) -> ${BIN_DIR}/auto-gdb"