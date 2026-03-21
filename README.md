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
# Add to PATH: export PATH="$HOME/.why/bin:$PATH"

# Configure Claude Code (one command)
why setup
```

That's it. Claude Code will now record reasoning before every edit automatically.

Use `why setup --project` to scope to a single project instead of installing globally.

## Commands

```bash
why blame <file>                  # Line-by-line reasoning
why history <file>                # Full edit history with reasoning
why history --related <file>      # + files changed together
why query "<question>"            # Ask anything about past decisions
why setup                         # Install globally
why setup --project               # Install per-project
why uninstall                     # Remove global config
why uninstall --project           # Remove per-project config
```

## How it works

`why` runs as an MCP server inside Claude Code. Before every edit, Claude calls
`record_why` with the reasoning. Pre/post hooks on Edit/Write/MultiEdit snapshot
the file, diff it, and map each changed line to its reasoning.

```
.why/
  objects/<hash>    # immutable reasoning entries (content-addressed)
  refs/<file>       # one hash per source line, like git blame
```

Claude can also query this data directly — when you ask "why did we change this?",
it uses `why_blame` and `why_history` MCP tools to answer with actual reasoning
instead of just commit messages.

## Requirements

- Claude Code
- `git` (optional, for commit hashes)

## Uninstall

```bash
why uninstall          # Remove global config
why uninstall --project # Remove per-project config
rm -rf .why/            # Optional: delete reasoning data
```
