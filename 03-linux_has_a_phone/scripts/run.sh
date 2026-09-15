#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

command -v dbus-run-session >/dev/null || {
  echo "D-Bus is required. Install the dbus package for your Linux distribution."
  exit 1
}

echo "Starting an isolated D-Bus session bus..."
echo

dbus-run-session -- bash -c '
  set -e
  go run ./cmd/dbus-demo server &
  SERVER_PID=$!
  trap "kill $SERVER_PID 2>/dev/null || true" EXIT
  go run ./cmd/dbus-demo client
'
