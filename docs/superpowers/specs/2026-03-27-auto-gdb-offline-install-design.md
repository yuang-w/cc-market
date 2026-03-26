---
name: auto-gdb-offline-install
description: Offline installation package for auto-gdb plugin via tar.gz
type: project
---

# auto-gdb Offline Installation Design

## Overview

Provide offline installation support for auto-gdb plugin in network-isolated environments. GitHub Action builds a tar.gz containing a mini marketplace structure. Users download, extract, and run install.sh which uses official Claude Code CLI commands to install the plugin, then unregisters the marketplace.

**Target Users**: Network-isolated machines that cannot access GitHub

**Success Criteria**:
- 3-step process: download → extract → run script
- Plugin immediately usable after installation
- Marketplace reference cleaned up after installation

## tar.gz Package Structure

```
auto-gdb-<version>-linux.tar.gz
└── auto-gdb-<version>-linux/
    ├── .claude-plugin/
    │   └── marketplace.json
    ├── plugins/
    │   └── auto-gdb/
    │       ├── plugin.json
    │       ├── .mcp.json
    │       ├── bin/
    │       │   ├── auto-gdb
    │       │   ├── auto-gdb-linux-amd64
    │       │   └── auto-gdb-linux-arm64
    │       ├── bridge/
    │       │   └── gdb_bridge.py
    │       └── skills/
    │           └── auto-gdb-investigation/
    │               └── SKILL.md
    ├── install.sh
    └── README.txt
```

## GitHub Action Workflow

| Step | Action |
|------|--------|
| 1 | Checkout code |
| 2 | Setup Go environment |
| 3 | Build amd64 and arm64 binaries |
| 4 | Assemble tar.gz directory structure |
| 5 | Create tar.gz and upload to GitHub Release |

## install.sh Logic

| Step | Command | Description |
|------|---------|-------------|
| 1 | `claude plugin marketplace add <path> --name <name>` | Register local marketplace |
| 2 | `claude plugin install auto-gdb@<name>` | Install plugin to cache |
| 3 | `claude plugin marketplace remove <name>` | Remove marketplace reference |

## User Operation Flow

```bash
# 1. Download on a machine with network access
#    Download auto-gdb-<version>-linux.tar.gz from GitHub Release

# 2. Copy to isolated machine

# 3. Extract
tar -xzf auto-gdb-<version>-linux.tar.gz

# 4. Install
cd auto-gdb-<version>-linux
./install.sh

# 5. (Optional) Cleanup
cd ..
rm -rf auto-gdb-<version>-linux
```

## Error Handling

| Scenario | Handling |
|----------|----------|
| `claude` command not found | Detect and prompt user to install Claude Code |
| Marketplace name conflict | Use random suffix to avoid collision |
| Plugin already installed | Prompt user to uninstall old version first |
| Installation interrupted | Cleanup already-added marketplace reference |

## File Source Mapping

| Path in tar.gz | Source |
|----------------|--------|
| `.claude-plugin/marketplace.json` | Generated in Action |
| `plugins/auto-gdb/plugin.json` | `plugins/auto-gdb/plugin.json` |
| `plugins/auto-gdb/.mcp.json` | `plugins/auto-gdb/.mcp.json` |
| `plugins/auto-gdb/bin/auto-gdb` | `plugins/auto-gdb/bin/auto-gdb` |
| `plugins/auto-gdb/bin/auto-gdb-linux-amd64` | Go build artifact |
| `plugins/auto-gdb/bin/auto-gdb-linux-arm64` | Go build artifact |
| `plugins/auto-gdb/bridge/gdb_bridge.py` | `plugins/auto-gdb/bridge/gdb_bridge.py` |
| `plugins/auto-gdb/skills/` | `plugins/auto-gdb/skills/` |
| `install.sh` | Created in this project |
| `README.txt` | Created in this project |