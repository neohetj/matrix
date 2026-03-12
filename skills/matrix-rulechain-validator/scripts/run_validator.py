#!/usr/bin/env python3
"""Run a repo-local Matrix rulechain validator from any workspace subdirectory."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path


def is_matrix_dsl_workspace(candidate: Path) -> bool:
    return (
        (candidate / "scripts" / "validate_rulechain_mappings.py").exists()
        and (candidate / "code" / "dsl").exists()
    )


def find_workspace_root(start: Path) -> Path | None:
    for candidate in [start, *start.parents]:
        if is_matrix_dsl_workspace(candidate):
            return candidate

        try:
            children = sorted(candidate.iterdir())
        except OSError:
            continue
        for child in children:
            if child.is_dir() and is_matrix_dsl_workspace(child):
                return child
    return None


def main(argv: list[str] | None = None) -> int:
    argv = argv if argv is not None else sys.argv[1:]
    cwd = Path.cwd().resolve()
    workspace_root = find_workspace_root(cwd)
    if workspace_root is None:
        print(
            "Matrix DSL workspace not found. Run this inside a repo containing code/dsl and scripts/validate_rulechain_mappings.py, or call the repo script directly.",
            file=sys.stderr,
        )
        return 2

    script = workspace_root / "scripts" / "validate_rulechain_mappings.py"
    command = [sys.executable, str(script), *argv]
    completed = subprocess.run(command, cwd=workspace_root)
    return completed.returncode


if __name__ == "__main__":
    raise SystemExit(main())
