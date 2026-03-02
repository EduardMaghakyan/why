#!/bin/bash
set -euo pipefail

INPUT=$(cat)

FILE=$(echo "$INPUT" | jq -r '
  .tool_input.path //
  .tool_input.file_path //
  (.tool_input.edits[0].path? // empty) //
  empty
' 2>/dev/null || true)

if [ -z "$FILE" ]; then exit 0; fi
if [[ "$FILE" == *.why ]]; then exit 0; fi
if [[ "$FILE" == *".claude/"* ]]; then exit 0; fi

WHY_FILE="${FILE}.why"
[ -f "$WHY_FILE" ] || exit 0

DIFF=""
if git rev-parse --git-dir > /dev/null 2>&1; then
  DIFF=$(git diff HEAD -- "$FILE" 2>/dev/null || true)
  if [ -z "$DIFF" ]; then
    DIFF=$(git diff --no-index /dev/null "$FILE" 2>/dev/null || true)
  fi
fi

if [ -n "$DIFF" ]; then
  DIFF_BLOCK='```diff\n'"$DIFF"'\n```\n\n---'
  perl -i -0pe "s|<!-- diff:pending -->|${DIFF_BLOCK}|" "$WHY_FILE" 2>/dev/null || true
else
  perl -i -0pe "s|<!-- diff:pending -->|---\n|" "$WHY_FILE" 2>/dev/null || true
fi
