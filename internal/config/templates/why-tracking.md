# Why Tracking

This project maintains a `.why` directory with reasoning behind every edit --
a decision journal that works like "git blame you can actually read."

## Before every edit, call `record_why`

**Before** calling `Edit`, `Write`, or `MultiEdit`, you MUST call the
`record_why` tool with:

- `file_path`: the file you are about to edit
- `reasoning`: a rich explanation covering:
  - **Why** this change is needed -- the actual problem being solved
  - **What** is changing at a high level
  - **Alternatives** you considered and why you rejected them
  - **Tradeoffs or risks** the change introduces

Hooks handle the rest automatically. You do not write `.why` files directly.

### Good reasoning:
```
Token refresh was racing with logout -- both read token state simultaneously
causing a double-refresh crash on slow connections. Added isRefreshing ref
flag to prevent re-entrant calls. Considered a mutex but it blocks the UI
thread. Tradeoff: a crashed refresh could leave the flag stuck; added 5s
timeout to auto-reset.
```

### Bad reasoning:
```
Fix bug
```

## Investigating code changes

When the user asks **why** code was changed, what the reasoning was behind a
change, or asks you to explain past edits — you MUST use the why-tracking
tools **before** or **instead of** relying solely on `git log`:

- `why_history` — show the full edit history with reasoning for a file
  (set `related: true` to see files changed together).
  Use this when asked about the history of changes to a file.
- `why_blame` — show line-by-line reasoning for any file.
  Use this when asked about specific lines or sections.

These tools provide the actual reasoning behind each edit (the "why"),
which git commit messages often lack. You may still use `git log` for
supplementary context, but always check why-tracking first.
