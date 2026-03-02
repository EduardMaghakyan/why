# Why Tracking

This project maintains a `.why` file next to every source file you edit —
a decision journal that works like "git blame you can actually read."

**Hooks handle everything automatically. You don't need to write .why files.**

## Before every edit, call `record_why`

**Before** calling `Edit`, `Write`, or `MultiEdit`, you MUST call the
`record_why` tool with:

- `file_path`: the file you are about to edit
- `reasoning`: a rich explanation covering:
  - **Why** this change is needed — the actual problem being solved
  - **What** is changing at a high level
  - **Alternatives** you considered and why you rejected them
  - **Tradeoffs or risks** the change introduces

### Good reasoning:
```
Token refresh was racing with logout — both read token state simultaneously
causing a double-refresh crash on slow connections. Added isRefreshing ref
flag to prevent re-entrant calls. Considered a mutex but it blocks the UI
thread. Tradeoff: a crashed refresh could leave the flag stuck; added 5s
timeout to auto-reset.
```

### Bad reasoning:
```
Fix bug
```

### Workflow example:

1. Call `record_why(file_path="src/auth/login.ts", reasoning="Token refresh was racing with logout...")`
2. Call `Edit(file_path="src/auth/login.ts", old_string=..., new_string=...)`

The pre-edit hook reads your recorded reasoning and writes the `.why` entry
with timestamp, commit hash, reasoning text, and diff.
