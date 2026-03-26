# auto-gdb Offline Installation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create tar.gz offline installation package for auto-gdb plugin, built and released via GitHub Action.

**Architecture:** GitHub Action builds Linux amd64/arm64 binaries, assembles a mini marketplace structure, and publishes tar.gz to GitHub Releases. Users download, extract, and run install.sh which uses official Claude CLI commands to register, install, and unregister the marketplace.

**Tech Stack:** Bash, GitHub Actions, Go build

---

## File Structure

| File | Purpose |
|------|---------|
| `plugins/auto-gdb/install.sh` | Installation script using Claude CLI |
| `.github/workflows/release-offline.yml` | GitHub Action to build and release tar.gz |

---

### Task 1: Create install.sh Script

**Files:**
- Create: `plugins/auto-gdb/install.sh`

- [ ] **Step 1: Write install.sh script**

```bash
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
```

- [ ] **Step 2: Make script executable**

Run: `chmod +x plugins/auto-gdb/install.sh`

- [ ] **Step 3: Commit**

```bash
git add plugins/auto-gdb/install.sh
git commit -m "feat(auto-gdb): add offline install script"
```

---

### Task 2: Create README.txt for Offline Package

**Files:**
- Create: `plugins/auto-gdb/README.txt`

- [ ] **Step 1: Write README.txt**

```
auto-gdb Offline Installation Package
=====================================

This package contains the auto-gdb plugin for Claude Code.
Designed for installation on machines without network access to GitHub.

Requirements:
- Claude Code CLI installed
- Linux (amd64 or arm64)

Installation:
1. Extract this package: tar -xzf auto-gdb-<version>-linux.tar.gz
2. Run the installer: cd auto-gdb-<version>-linux && ./install.sh
3. (Optional) Delete the extracted directory after installation

After installation, the plugin files are stored in:
~/.claude/plugins/cache/

For usage documentation, see:
https://github.com/yuang-w/cc-market/tree/main/plugins/auto-gdb
```

- [ ] **Step 2: Commit**

```bash
git add plugins/auto-gdb/README.txt
git commit -m "docs(auto-gdb): add README for offline package"
```

---

### Task 3: Create GitHub Action for Release

**Files:**
- Create: `.github/workflows/release-offline.yml`

- [ ] **Step 1: Create workflows directory**

Run: `mkdir -p .github/workflows`

- [ ] **Step 2: Write release workflow**

```yaml
name: Release Offline Package

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  build-and-release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Get version
        id: version
        run: |
          # Extract version from plugin.json
          VERSION=$(grep '"version"' plugins/auto-gdb/plugin.json | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
          echo "version=$VERSION" >> $GITHUB_OUTPUT

      - name: Build binaries
        run: |
          cd plugins/auto-gdb/src
          make build-all

      - name: Assemble package
        run: |
          VERSION="${{ steps.version.outputs.version }}"
          PKG_DIR="auto-gdb-${VERSION}-linux"

          mkdir -p "${PKG_DIR}/.claude-plugin"
          mkdir -p "${PKG_DIR}/plugins/auto-gdb"

          # Marketplace manifest
          cat > "${PKG_DIR}/.claude-plugin/marketplace.json" << EOF
          {
            "name": "auto-gdb-offline",
            "description": "auto-gdb offline installation package",
            "owner": {
              "name": "yuang-w"
            },
            "plugins": [
              {
                "name": "auto-gdb",
                "description": "GDB-based debugging MCP server with investigation skill for production coredump analysis",
                "version": "${VERSION}",
                "source": "./plugins/auto-gdb",
                "author": {
                  "name": "yuang-w"
                }
              }
            ]
          }
          EOF

          # Plugin files
          cp plugins/auto-gdb/plugin.json "${PKG_DIR}/plugins/auto-gdb/"
          cp plugins/auto-gdb/.mcp.json "${PKG_DIR}/plugins/auto-gdb/"
          cp -r plugins/auto-gdb/bin "${PKG_DIR}/plugins/auto-gdb/"
          cp -r plugins/auto-gdb/bridge "${PKG_DIR}/plugins/auto-gdb/"
          cp -r plugins/auto-gdb/skills "${PKG_DIR}/plugins/auto-gdb/"
          cp -r plugins/auto-gdb/hooks "${PKG_DIR}/plugins/auto-gdb/"

          # Install script and README
          cp plugins/auto-gdb/install.sh "${PKG_DIR}/"
          cp plugins/auto-gdb/README.txt "${PKG_DIR}/"

          # Create tar.gz
          tar -czvf "${PKG_DIR}.tar.gz" "${PKG_DIR}"

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: auto-gdb-*.tar.gz
          generate_release_notes: true
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release-offline.yml
git commit -m "ci: add GitHub Action for offline package release"
```

---

### Task 4: Update CLAUDE.md Documentation

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Add offline installation section to CLAUDE.md**

Add after the "Auto-GDB Plugin Notes" section:

```markdown
## Offline Installation

For machines without network access to GitHub, auto-gdb can be installed from a tar.gz package:

1. Download `auto-gdb-<version>-linux.tar.gz` from [GitHub Releases](https://github.com/yuang-w/cc-market/releases)
2. Copy to the target machine
3. Extract and install:
   ```bash
   tar -xzf auto-gdb-<version>-linux.tar.gz
   cd auto-gdb-<version>-linux
   ./install.sh
   ```
4. (Optional) Delete the extracted directory

The installation uses official Claude CLI commands (`claude plugin marketplace add/install/remove`).
```

- [ ] **Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add offline installation instructions"
```

---

### Task 5: Update README.md

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add offline installation section to README.md**

Add after line 17 (after the `/plugin install auto-gdb` code block):

```markdown

## Offline Installation

For network-isolated environments, download the offline package from [GitHub Releases](https://github.com/yuang-w/cc-market/releases):

```bash
tar -xzf auto-gdb-<version>-linux.tar.gz
cd auto-gdb-<version>-linux
./install.sh
```
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add offline installation to README"
```

---

## Self-Review

**1. Spec coverage:**
- [x] tar.gz structure defined in Task 3
- [x] GitHub Action workflow in Task 3
- [x] install.sh logic in Task 1
- [x] User operation flow documented in Task 4, 5
- [x] Error handling in Task 1 (claude command check, trap for cleanup)

**2. Placeholder scan:** No TBD, TODO, or placeholder patterns found.

**3. Type consistency:** N/A - no type definitions across tasks.