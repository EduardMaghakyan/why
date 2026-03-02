#!/bin/bash
# why-tracking/setup.sh
#
# Installs .why shadow file tracking into the current working directory.
# Run from inside any project:
#
#   bash /path/to/why-tracking/setup.sh
#
set -e

WHY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(pwd)"

# ── Checks ────────────────────────────────────────────────────────────────────

if [ "$PROJECT_DIR" = "$WHY_DIR" ]; then
  echo "⚠️  Run this from your project root, not from inside why-tracking."
  echo "   cd /your/project && bash $WHY_DIR/setup.sh"
  exit 1
fi

if ! command -v jq &>/dev/null; then
  echo "⚠️  jq is required but not installed."
  echo "   brew install jq   or   apt install jq"
  exit 1
fi

echo "Installing why-tracking into: $PROJECT_DIR"
echo ""

# ── Copy hooks ────────────────────────────────────────────────────────────────

mkdir -p "$PROJECT_DIR/.claude/hooks"
cp "$WHY_DIR/.claude/hooks/pre-edit.sh"  "$PROJECT_DIR/.claude/hooks/pre-edit.sh"
cp "$WHY_DIR/.claude/hooks/post-edit.sh" "$PROJECT_DIR/.claude/hooks/post-edit.sh"
chmod +x "$PROJECT_DIR/.claude/hooks/pre-edit.sh"
chmod +x "$PROJECT_DIR/.claude/hooks/post-edit.sh"
echo "✓ Hooks installed"

# ── Copy MCP server ──────────────────────────────────────────────────────────

mkdir -p "$PROJECT_DIR/.claude/mcp"
cp "$WHY_DIR/.claude/mcp/why-server.py" "$PROJECT_DIR/.claude/mcp/why-server.py"
echo "✓ MCP server installed"

# ── Ensure uvx is available ───────────────────────────────────────────────────

if ! command -v uvx &>/dev/null; then
  echo "  Installing uv (needed to run MCP server)..."
  curl -LsSf https://astral.sh/uv/install.sh | sh
  export PATH="$HOME/.local/bin:$PATH"
  echo "✓ Installed uv"
fi

# ── Create .mcp.json ─────────────────────────────────────────────────────────

MCP_JSON="$PROJECT_DIR/.mcp.json"
if [ ! -f "$MCP_JSON" ]; then
  cat > "$MCP_JSON" << 'MCPEOF'
{
  "mcpServers": {
    "why-tracker": {
      "command": "uvx",
      "args": ["fastmcp", "run", ".claude/mcp/why-server.py"]
    }
  }
}
MCPEOF
  echo "✓ Created .mcp.json"
elif grep -q "why-tracker" "$MCP_JSON"; then
  echo "✓ .mcp.json already configured"
else
  echo ""
  echo "⚠️  .mcp.json already exists. Add this server manually:"
  echo '  "why-tracker": { "command": "uvx", "args": ["fastmcp", "run", ".claude/mcp/why-server.py"] }'
fi

# ── Copy Claude instruction file ──────────────────────────────────────────────

cp "$WHY_DIR/.claude/why-tracking.md" "$PROJECT_DIR/.claude/why-tracking.md"
echo "✓ Instruction file installed"

# ── Patch CLAUDE.md ───────────────────────────────────────────────────────────

CLAUDE_MD="$PROJECT_DIR/CLAUDE.md"
if [ ! -f "$CLAUDE_MD" ]; then
  echo "@.claude/why-tracking.md" > "$CLAUDE_MD"
  echo "✓ Created CLAUDE.md"
elif ! grep -q "why-tracking.md" "$CLAUDE_MD"; then
  echo "" >> "$CLAUDE_MD"
  echo "@.claude/why-tracking.md" >> "$CLAUDE_MD"
  echo "✓ Appended to existing CLAUDE.md"
else
  echo "✓ CLAUDE.md already includes why-tracking"
fi

# ── Patch .claude/settings.json ───────────────────────────────────────────────

SETTINGS="$PROJECT_DIR/.claude/settings.json"
FRAGMENT="$WHY_DIR/.claude/settings.fragment.json"

if [ ! -f "$SETTINGS" ]; then
  cp "$FRAGMENT" "$SETTINGS"
  echo "✓ Created .claude/settings.json"
else
  # Check if hooks are already present
  if grep -q "pre-edit.sh" "$SETTINGS"; then
    echo "✓ .claude/settings.json already configured"
  else
    echo ""
    echo "⚠️  You already have a .claude/settings.json."
    echo "   Merge this block into it manually:"
    echo ""
    cat "$FRAGMENT"
    echo ""
  fi
fi

# ── Patch ignore files ────────────────────────────────────────────────────────

for f in .eslintignore .prettierignore; do
  IGNORE_FILE="$PROJECT_DIR/$f"
  if [ -f "$IGNORE_FILE" ] && ! grep -q "\*.why" "$IGNORE_FILE"; then
    echo "*.why" >> "$IGNORE_FILE"
    echo "✓ Added *.why to $f"
  fi
done

# ── Done ──────────────────────────────────────────────────────────────────────

echo ""
echo "✓ Done. Shadow .why files will appear automatically as Claude edits your code."
echo ""
echo "Useful commands:"
echo "  git log -p src/foo.ts.why         # decision history for a file"
echo "  grep -r 'why' **/*.why            # search reasoning across project"
echo "  git diff HEAD~1 -- '*.why'        # all reasoning from last commit"
echo ""
echo "To uninstall:"
echo "  bash $WHY_DIR/uninstall.sh"
