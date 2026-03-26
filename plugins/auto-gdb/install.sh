#!/bin/bash
set -e

# Offline installer for auto-gdb plugin
# Usage: ./install.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MARKETPLACE_NAME="auto-gdb-offline-$$"

# Check if claude command exists
if ! command -v claude &> /dev/null; then
    echo "Error: 'claude' command not found."
    echo "Please install Claude Code first: https://docs.anthropic.com/en/docs/claude-code"
    exit 1
fi

echo "Installing auto-gdb plugin..."

# 1. Add local marketplace
echo "Adding local marketplace..."
claude plugin marketplace add "$SCRIPT_DIR" --name "$MARKETPLACE_NAME"

# 2. Install plugin (trap to cleanup on failure)
cleanup() {
    echo "Cleaning up marketplace reference..."
    claude plugin marketplace remove "$MARKETPLACE_NAME" 2>/dev/null || true
}
trap cleanup EXIT

echo "Installing plugin..."
claude plugin install "auto-gdb@$MARKETPLACE_NAME"

# 3. Remove marketplace reference (plugin is now in cache)
echo "Removing marketplace reference..."
claude plugin marketplace remove "$MARKETPLACE_NAME"
trap - EXIT

echo ""
echo "✓ auto-gdb installed successfully!"
echo "  The plugin is now available in Claude Code."
echo "  You can safely delete this directory: $SCRIPT_DIR"
