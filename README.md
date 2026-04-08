# Claude Code Marketplace

Claude Code plugins by yuang-w.

## Usage

Add this marketplace:

```bash
/plugin marketplace add https://github.com/yuang-w/cc-market.git
```

Then install the plugin:

```bash
/plugin install autogdb
```

## Offline Installation

For network-isolated environments, install from a tar.gz package:

1. Download `autogdb-<version>-linux.tar.gz` from [GitHub Releases](https://github.com/yuang-w/cc-market/releases) on a connected machine
2. Copy to the target machine
3. Extract and install:
   ```bash
   tar -xzf autogdb-<version>-linux.tar.gz
   cd autogdb-<version>-linux
   ./install.sh
   ```

## What's Included

- **MCP server** (`autogdb`) - GDB control via MCP
- **Investigation skill** (`/autogdb-investigation`) - Production debugging workflow
- **GDB bridge script** - For socket mode connections

## Versioning

- **Plugin version**: version in `plugins/autogdb/plugin.json` and `.claude-plugin/marketplace.json`

**When releasing a new version:**
1. Run `./release.sh` to build binaries for both architectures
2. Commit the updated binaries in `plugins/autogdb/bin/`
3. Bump the plugin versions in **both** `plugins/autogdb/plugin.json` **and** `.claude-plugin/marketplace.json`
4. Create the corresponding git tag and push it

Binaries are included in the repository, so no separate release assets are needed.