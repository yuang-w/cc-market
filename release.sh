#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Building auto-gdb binaries..."
cd "${SCRIPT_DIR}/plugins/auto-gdb/src"
make build-all

echo "Build complete."
ls -la ../bin/auto-gdb-Linux-*