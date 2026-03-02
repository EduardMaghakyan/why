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
TIMESTAMP=$(date +"%Y-%m-%d %H:%M")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "no-git")

# Read reasoning from MCP server's pending file
WHY_PENDING_DIR="/tmp/.why-pending"
ABS_FILE="$(cd "$(dirname "$FILE")" 2>/dev/null && pwd)/$(basename "$FILE")" 2>/dev/null || ABS_FILE="$(pwd)/$FILE"
KEY=$(echo -n "$ABS_FILE" | shasum -a 256 | cut -c1-16)
PENDING_FILE="$WHY_PENDING_DIR/$KEY"

DESCRIPTION=""
if [ -f "$PENDING_FILE" ]; then
  DESCRIPTION=$(cat "$PENDING_FILE")
  rm -f "$PENDING_FILE"
fi

{
  echo ""
  echo "## $TIMESTAMP | $COMMIT"
  echo ""
  if [ -n "$DESCRIPTION" ]; then
    echo "$DESCRIPTION"
  else
    echo "_(no reasoning provided)_"
  fi
  echo ""
  echo "<!-- diff:pending -->"
  echo ""
} >> "$WHY_FILE"
