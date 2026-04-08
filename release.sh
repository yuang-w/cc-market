#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Building autogdb binaries..."
cd "${SCRIPT_DIR}/plugins/autogdb/src"
make build-all

echo "Build complete."
ls -la ../bin/autogdb-Linux-*