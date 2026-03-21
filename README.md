# why

`git blame` tells you *what* changed. `why` tells you *why*.

Every time Claude Code edits your code, `why` captures the reasoning — the problem,
the alternatives considered, the tradeoffs. Then you can query it.

```bash
$ why query "what race conditions have we fixed?"

Based on the reasoning journal, there was a token refresh race condition
where refresh() and logout() both read token state simultaneously, causing
a double-refresh crash on slow connections. An isRefreshing ref flag was
added to prevent re-entrant calls, with a 5s timeout to auto-reset.
```

Ask a question in plain English. Get the actual reasoning back — not commit messages.

## See reasoning on every line

```
src/auth/login.ts

── a3f9c2b: Token refresh racing with logout — added isRefreshing guard ──
   1 │ import { useRef } from 'react'
   2 │ const isRefreshing = useRef(false)
   3 │ if (!isRefreshing.current) refresh()
```

Like `git blame`, but instead of "who wrote this line", you get "why does this line exist".

## See the full decision history

```bash
$ why history --related src/auth/login.ts

## 2026-03-19 14:32 | a3f9c2b

Token refresh racing with logout — refresh() and logout() both read token
state simultaneously, causing a double-refresh crash on slow connections.
Added an isRefreshing ref flag to prevent re-entrant calls, with a 5s
timeout to auto-reset.

  Also changed:
    src/auth/logout.ts
    src/hooks/useSession.ts
```

Every edit, with full reasoning and related files — a decision journal that writes itself.

## Getting Started

```bash
# Install
go install github.com/eduardmaghakyan/why@latest

# Or build from source
git clone https://github.com/eduardmaghakyan/why.git && cd why
make install
# Add to PATH for CLI usage: export PATH="$HOME/.why/bin:$PATH"

# Configure Claude Code (one command)
why setup
```

That's it. `why setup` installs hooks that automatically track every edit.
Hooks use absolute paths, so no PATH configuration is needed for them to work.

Use `why setup --project` to scope to a single project instead of installing globally.

## Commands

```bash
why record <file> '<reasoning>'   # Record reasoning before an edit
why blame <file>                  # Line-by-line reasoning
why history <file>                # Full edit history with reasoning
why history --related <file>      # + files changed together
why query "<question>"            # Ask anything about past decisions
why setup                         # Install globally
why setup --project               # Install per-project
why setup --mcp                   # Install with MCP server (optional)
why uninstall                     # Remove global config
why uninstall --project           # Remove per-project config
```

## How it works

Before every edit, Claude runs `why record` to capture the reasoning. Hooks on
Edit/Write/MultiEdit automatically snapshot the file, diff it after the edit,
and map each changed line to its reasoning hash.

```
Claude runs: why record src/main.go 'Fix race condition in token refresh'
Claude runs: Edit(src/main.go, ...)
  → pre-hook: snapshots file, reads reasoning hash
  → [edit happens]
  → post-hook: diffs old vs new, updates .why/refs/src/main.go
```

Turn-based grouping tracks which edits belong together: a `UserPromptSubmit` hook
marks the start of each turn, and all edits within that turn share a turn ID.
This powers `--related` — finding files that were changed together.

If `why record` wasn't called (e.g., Claude skipped it), a transcript fallback
reads the reasoning from the conversation history automatically.

```
.why/
  objects/<hash>    # immutable reasoning entries (content-addressed)
  refs/<file>       # one hash per source line, like git blame
```

### MCP mode (optional)

For a structured tool interface instead of Bash commands:

```bash
why setup --mcp
```

This registers an MCP server that exposes `record_why`, `why_blame`, and
`why_history` as tools Claude can call directly.

## Requirements

- Claude Code
- `git` (optional, for commit hashes)

## Uninstall

```bash
why uninstall          # Remove global config
why uninstall --project # Remove per-project config
rm -rf .why/            # Optional: delete reasoning data
```
