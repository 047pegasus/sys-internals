#!/usr/bin/env bash
set -euo pipefail

if ! command -v strace >/dev/null 2>&1; then
  echo "strace is required: install it with your distro package manager." >&2
  exit 1
fi

go build -o malloc-demo ./cmd/malloc-demo
exec strace -f -e trace=brk,mmap,munmap,madvise ./malloc-demo
