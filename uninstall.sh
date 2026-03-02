#!/bin/bash
# why-tracking/uninstall.sh
#
# Removes why-tracking from the current project.
# Run from inside the project:
#
#   bash /path/to/why-tracking/uninstall.sh
#
set -e

PROJECT_DIR="$(pwd)"

echo "Removing why-tracking from: $PROJECT_DIR"
echo ""

rm -f "$PROJECT_DIR/.claude/hooks/pre-edit.sh"
rm -f "$PROJECT_DIR/.claude/hooks/post-edit.sh"
rm -f "$PROJECT_DIR/.claude/why-tracking.md"
rm -f "$PROJECT_DIR/.claude/mcp/why-server.py"
rmdir "$PROJECT_DIR/.claude/mcp" 2>/dev/null || true
echo "✓ Hooks, MCP server, and instruction file removed"

# ── Remove MCP config ───────────────────────────────────────────────────────
MCP_JSON="$PROJECT_DIR/.mcp.json"
if [ -f "$MCP_JSON" ] && grep -q "why-tracker" "$MCP_JSON"; then
  if [ "$(jq '.mcpServers | keys | length' "$MCP_JSON" 2>/dev/null)" = "1" ]; then
    rm -f "$MCP_JSON"
    echo "✓ Removed .mcp.json"
  else
    echo "⚠️  .mcp.json has other servers. Remove the 'why-tracker' entry manually."
  fi
fi

# ── Clean up temp files ──────────────────────────────────────────────────────
rm -rf /tmp/.why-pending

CLAUDE_MD="$PROJECT_DIR/CLAUDE.md"
if [ -f "$CLAUDE_MD" ]; then
  # Remove the @include line and any blank line before it
  perl -i -0pe 's/\n*\n@\.claude\/why-tracking\.md//g' "$CLAUDE_MD"
  echo "✓ Removed from CLAUDE.md"
fi

echo ""
echo "✓ Done. .why files are left in place — delete them manually if you want."
