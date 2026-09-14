#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

LOG="memory_log.csv"
: > "$LOG"

go run ./cmd/fork-demo | tee >(grep '^FORK_STATE,' > "$LOG")

if command -v python3 >/dev/null 2>&1; then
    python3 scripts/plot_memory.py "$LOG"
else
    echo "python3 not found; skipping plot generation."
fi
