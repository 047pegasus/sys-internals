# Linux `fork()` + Copy-on-Write — Go Demonstration

A small Linux-only Go project that makes the most interesting part of `fork()` visible: **copy-on-write (COW)**.

## What the demo shows

The parent process reserves and touches 32 MB of anonymous memory. It then calls `fork()`.

Linux does **not** immediately make a complete physical copy of those 32 MB. Instead, the parent and child initially refer to the same physical pages, with page-table state arranged so that a write can trigger a copy.

The child then writes to half of the pages. Those writes are where COW becomes visible: the modified pages become private to the child while the parent keeps its original pages.

The program reads `/proc/self/status` and `/proc/self/smaps` and logs:

- `VmRSS` — resident memory
- `Private_Dirty` — private modified pages
- `Shared_Dirty` — modified pages currently shared
- `Pss` — proportional set size

A Python script turns those observations into a plot.

## Why C is used inside the Go project

Do not implement `fork()` as a raw syscall from ordinary Go code and then continue running normal Go code in the child. The Go runtime has threads and internal state, so forking a Go process and continuing arbitrary runtime operations in the child is unsafe.

This project therefore keeps the `fork()` experiment in a small C function called through cgo. The child performs only simple, controlled work and exits with `_exit()`.

## Requirements

Linux, Go, a C compiler, and Python 3 with matplotlib for the optional graph.

On Debian/Ubuntu:

```bash
sudo apt install build-essential python3 python3-matplotlib
```

## Run

```bash
./scripts/run.sh
```

Or just run the Go program:

```bash
go run ./cmd/fork-demo
```

The run script creates `memory_log.csv` and, when matplotlib is available, `fork_cow_memory.png`.

## Expected story

The important sequence is:

```text
Parent allocates + touches memory
          ↓
        fork()
          ↓
 Parent + child initially share physical pages
          ↓
 Child writes to some pages
          ↓
 Kernel copies only the pages being modified
          ↓
 Parent keeps its original pages
 Child gets private copies of modified pages
```

This is why the classic mental model of `fork()` as “copy the entire process immediately” is misleading on modern Linux. The virtual address spaces are separate after `fork()`, but physical memory is initially shared wherever copy-on-write permits it.

## Reading Material & Links:
https://www.microsoft.com/en-us/security/blog/2026/05/01/cve-2026-31431-copy-fail-vulnerability-enables-linux-root-privilege-escalation/

https://man7.org/linux/man-pages/man2/fork.2.html

https://www.kernel.org/doc/gorman/html/understand/understand007.html

https://kernel-internals.org/mm/cow/
