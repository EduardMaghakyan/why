# why

A content-addressable decision journal for Claude Code. Captures the *reasoning*
behind every edit — not just what changed, but why.

Think `git blame`, but for the "why".

## Getting Started

### 1. Install the CLI

```bash
# From source (requires Go)
go install github.com/eduardmaghakyan/why@latest

# Or download a binary
curl -fsSL https://raw.githubusercontent.com/eduardmaghakyan/why/main/install.sh | sh

# Or build from source
git clone https://github.com/eduardmaghakyan/why.git
cd why
make install
# Then add to PATH: export PATH="$HOME/.why/bin:$PATH"
```

### 2. Configure Claude Code

```bash
why setup
```

This installs **globally** so why-tracking works in every project:
- `~/.claude.json` — registers the MCP server with Claude Code
- `~/.claude/settings.json` — hooks for Edit/Write/MultiEdit + auto-allows all why-tracker tools
- `~/.claude/why-tracking.md` — instructions for Claude (included via `@.claude/why-tracking.md` in CLAUDE.md)

The `.why/` data directory and `.gitignore` entry are created per-project automatically.

### Per-project only

To scope everything to a single project instead of installing globally:

```bash
why setup --project
```

This creates `.mcp.json`, `.claude/settings.json`, `.claude/why-tracking.md`,
and patches `CLAUDE.md` — all within the current project directory.

## How it works

```
Claude → record_why(file, reasoning) → object stored in .why/objects/
Claude → Edit(file, ...) →
  pre-hook: snapshots file + reads reasoning hash
  [edit happens]
  post-hook: diffs old vs new → updates .why/refs/<file> line-by-line
```

Every reasoning entry is stored as an immutable, content-addressed object.
A refs file maps each source line to its reasoning — like git blame, but for decisions.

### MCP tools

The why-tracker MCP server exposes three tools to Claude:

| Tool | Purpose |
|------|---------|
| `record_why` | Record reasoning before an edit (called automatically before Edit/Write/MultiEdit) |
| `why_blame` | Show line-by-line reasoning for a file |
| `why_history` | Show edit history with full reasoning for a file (optionally with related files) |

When you ask Claude "why did we change this file?", it uses `why_history` and
`why_blame` to answer with the actual reasoning — not just commit messages.

## Storage

```
.why/
  objects/<2char>/<hash>   # immutable reasoning: {"ts", "commit", "reasoning"}
  refs/<source-path>       # one hash per line, aligned with source
```

## Commands

```bash
# Line-by-line reasoning (like git blame)
why blame src/auth/login.ts

# Edit history for a file
why history src/auth/login.ts

# Edit history with related files changed together
why history --related src/auth/login.ts

# Ask a question about reasoning using Claude
why query "why did we add the refresh guard?"

# Install globally (default)
why setup

# Install per-project only
why setup --project

# Remove global config
why uninstall

# Remove per-project config
why uninstall --project
```

### Example blame output

```
src/auth/login.ts

── a3f9c2b: Token refresh racing with logout — added isRefreshing guard ──
   1 │ import { useRef } from 'react'
   2 │ const isRefreshing = useRef(false)
   3 │ if (!isRefreshing.current) refresh()
```

### Example query

```bash
$ why query "what race conditions have we fixed?"
Based on the reasoning journal, there was a token refresh race condition
where refresh() and logout() both read token state simultaneously, causing
a double-refresh crash on slow connections. An isRefreshing ref flag was
added to prevent re-entrant calls, with a 5s timeout to auto-reset.
```

## Requirements

- Claude Code
- `git` (optional, for commit hashes in reasoning entries)

## Uninstall

```bash
# Remove global config (default)
why uninstall

# Remove per-project config
why uninstall --project

# Optional: delete reasoning data
rm -rf .why/
```
