#!/usr/bin/env python3
import json
import subprocess
import sys
from pathlib import Path

root = Path(__file__).resolve().parents[2]
event = json.load(sys.stdin)

# Avoid an endless continuation when a check cannot be fixed in this environment.
if event.get("stop_hook_active"):
    print("{}")
    sys.exit(0)

status = subprocess.run(
    ["git", "status", "--porcelain", "--untracked-files=all", "--", "frontend/"],
    cwd=root,
    capture_output=True,
    text=True,
)
if status.returncode:
    print(json.dumps({"decision": "block", "reason": f"Could not check frontend changes: {status.stderr}. Report this limitation."}))
    sys.exit(0)
if not status.stdout:
    print("{}")
    sys.exit(0)

result = subprocess.run(
    ["pnpm", "exec", "biome", "check", "."],
    cwd=root / "frontend",
    capture_output=True,
    text=True,
)
if result.returncode:
    reason = f"Frontend formatting and lint checks failed (exit {result.returncode}). Fix issues related to this work if possible; otherwise report the failures and limitations.\n\n{result.stdout}{result.stderr}"
    print(json.dumps({"decision": "block", "reason": reason}))
else:
    print("{}")
