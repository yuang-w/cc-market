# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is a Claude Code plugin marketplace containing plugins published by yuang-w. Users add this marketplace to Claude Code and install plugins from it.

## Plugin Structure

Each plugin lives in `plugins/<name>/` with:

- `plugin.json` - Plugin metadata (name, version, description, homepage, repository)
- `.mcp.json` - MCP server configuration (uses `${CLAUDE_PLUGIN_ROOT}` for paths)
- `skills/` - Optional skill directories with `SKILL.md` files
- `scripts/` - Optional install scripts (referenced by `postInstall` in plugin.json)
- `bridge/` - Optional supporting files (e.g., GDB bridge script)

## Marketplace Configuration

`.claude-plugin/marketplace.json` defines the marketplace:
- `name`, `description`, `owner` - Marketplace identity
- `plugins` - Array of available plugins with `source` paths pointing to plugin directories

## Adding a New Plugin

1. Create `plugins/<name>/` directory
2. Add `plugin.json` with required fields
3. Add `.mcp.json` if the plugin includes an MCP server
4. Add skills in `skills/<skill-name>/SKILL.md` if applicable
5. Register the plugin in `.claude-plugin/marketplace.json`

## Auto-GDB Plugin Notes

The auto-gdb plugin provides GDB-based debugging for production coredump analysis. Key components:

- **MCP server** (`bin/auto-gdb`): Binary provided externally at install time via `AUTO_GDB_BINARY_URL` or `AUTO_GDB_BINARY_PATH`
- **Socket mode bridge** (`bridge/gdb_bridge.py`): Python script loaded into GDB for remote control via Unix socket
- **Investigation skill**: Hypothesis-driven debugging workflow for coredumps

The plugin supports two GDB modes:
- **Subprocess mode**: MCP spawns and controls GDB directly
- **Socket mode**: User runs GDB with bridge loaded, MCP connects via socket

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

The install.sh script handles all setup using official Claude CLI commands.
