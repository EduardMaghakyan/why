# why-tracking

Automatically maintains a `.why` shadow file next to every file Claude Code
edits — a decision journal that captures the *reasoning* behind each change,
not just the diff.

Works like `git blame` but for the "why".

## Install

Clone or download this repo anywhere on your machine, then run from inside
any project you want to track:

```bash
cd /your/project
bash /path/to/why-tracking/setup.sh
```

## What you get

For every file Claude edits, a companion `.why` file is created alongside it:

```
src/auth/login.ts
src/auth/login.ts.why   ← decision journal
```

Each entry looks like:

```
## 2024-01-15 14:32 | a3f9c2b

Token refresh was racing with logout — both read token state simultaneously
causing a double-refresh crash on slow connections. Added isRefreshing ref
flag to prevent re-entrant calls. Considered a mutex but it blocks the UI
thread. Tradeoff: a crashed refresh could leave the flag stuck; added 5s
timeout to auto-reset.

​```diff
+ const isRefreshing = useRef(false)
- if (token.expired) refresh()
+ if (token.expired && !isRefreshing.current) refresh()
​```

---
```

## Useful commands

```bash
# Full decision history for one file
git log -p src/auth/login.ts.why

# Search reasoning across the whole project
grep -r "race condition" **/*.why

# All reasoning added in the last commit
git diff HEAD~1 -- '*.why'

# Pair with git blame — grab a hash, look it up
git blame src/auth/login.ts   # → find hash a3f9c2b
grep -A 20 "a3f9c2b" src/auth/login.ts.why
```

## Uninstall

```bash
cd /your/project
bash /path/to/why-tracking/uninstall.sh
```

## Requirements

- Claude Code
- `jq` (`brew install jq` or `apt install jq`)
- `uv` (auto-installed by setup.sh, or `curl -LsSf https://astral.sh/uv/install.sh | sh`)
- `git` (optional but recommended for diffs)
