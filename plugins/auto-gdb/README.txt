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
