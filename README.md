# why

A content-addressable decision journal for Claude Code. Captures the *reasoning*
behind every edit — not just what changed, but why.

Think `git blame`, but for the "why".

## Install

```bash
# From source
go install github.com/eduardmaghakyan/why@latest

# Or download a binary
curl -fsSL https://raw.githubusercontent.com/eduardmaghakyan/why/main/install.sh | sh
```

## Setup

Run inside any project:

```bash
why setup
```

This creates:
- `.mcp.json` — registers the MCP server with Claude Code
- `.claude/settings.json` — hooks for Edit/Write/MultiEdit
- `.claude/why-tracking.md` — instructions for Claude
- `CLAUDE.md` — includes the instruction file
- `.gitignore` — ignores `.why/`

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

# Install into a project
why setup

# Remove from a project
why uninstall
```

### Example blame output

```
   1                                                             import { useRef } from 'react'
   2  a3f9c2b  Token refresh racing with logout — added guard    const isRefreshing = useRef(false)
   3  a3f9c2b  Token refresh racing with logout — added guard    if (!isRefreshing.current) refresh()
```

## Search reasoning

```bash
grep -r "race condition" .why/objects/
```

## Requirements

- Claude Code
- `git` (optional, for commit hashes in reasoning entries)

## Uninstall

```bash
why uninstall
rm -rf .why/  # optional: delete reasoning data
```
