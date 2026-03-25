# Claude Code Marketplace

Claude Code plugins by yuang-w.

## Usage

Add this marketplace:

```bash
/plugin marketplace add https://github.com/yuang-w/cc-market.git
```

Then install the plugin:

```bash
/plugin install auto-gdb
```

## What's Included

- **MCP server** (`auto-gdb`) - GDB control via MCP
- **Investigation skill** (`/auto-gdb-investigation`) - Production debugging workflow
- **GDB bridge script** - For socket mode connections

## Versioning

- **Plugin version**: version in `plugins/auto-gdb/plugin.json` and `.claude-plugin/marketplace.json`

**When releasing a new version:**
1. Run `./release.sh` to build binaries for both architectures
2. Commit the updated binaries in `plugins/auto-gdb/bin/`
3. Bump the plugin versions in `marketplace.json` and `plugins/auto-gdb/plugin.json`
4. Create the corresponding git tag and push it

Binaries are included in the repository, so no separate release assets are needed.