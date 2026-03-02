#!/usr/bin/env python3
"""MCP server that records reasoning before code edits."""

import hashlib
import os

from fastmcp import FastMCP

mcp = FastMCP("why-tracker")

WHY_DIR = "/tmp/.why-pending"


@mcp.tool
def record_why(file_path: str, reasoning: str) -> str:
    """Record the reasoning for an upcoming file edit.

    Call this BEFORE every Edit, Write, or MultiEdit to capture
    why the change is being made. The reasoning will be automatically
    attached to the .why shadow file by the pre-edit hook.

    Args:
        file_path: The file that is about to be edited (relative or absolute).
        reasoning: Why this change is needed - the problem, alternatives
                   considered, and tradeoffs.
    """
    os.makedirs(WHY_DIR, exist_ok=True)
    key = hashlib.sha256(os.path.abspath(file_path).encode()).hexdigest()[:16]
    pending_file = os.path.join(WHY_DIR, key)
    with open(pending_file, "w") as f:
        f.write(reasoning)
    return f"Reasoning recorded for {file_path}. Proceed with your edit."


if __name__ == "__main__":
    mcp.run()
