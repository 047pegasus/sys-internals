#!/usr/bin/env python3
"""
plot_memory.py
==============

Turns the memory_log.csv produced by `go run ./cmd/malloc-demo` into two
plain-language graphs:

  1. requested_vs_allocated.png
     For every allocation: how big was the request vs. how big a block did
     the allocator actually hand back? (Allocators round up -- this makes
     that visible.)

  2. freed_vs_held.png
     Over the life of the program: how much memory did we *ask* to free,
     compared to how much memory the operating system actually got back?
     The gap between the two lines is memory the allocator is quietly
     holding onto for itself instead of returning it to the free pool.

Usage:
    python3 scripts/plot_memory.py [path/to/memory_log.csv] [output_dir]

Defaults: memory_log.csv (in the current directory), output_dir="."

Requires: matplotlib (pip install matplotlib)
"""

import csv
import os
import sys


def load_rows(path):
    rows = []
    with open(path, newline="") as f:
        reader = csv.DictReader(f)
        for r in reader:
            rows.append(
                {
                    "step": int(r["step"]),
                    "section": r["section"],
                    "event": r["event"],
                    "label": r["label"],
                    "requested_bytes": int(r["requested_bytes"]),
                    "allocated_bytes": int(r["allocated_bytes"]),
                    "rss_kb": int(r["rss_kb"]),
                }
            )
    return rows


def to_kb(n_bytes):
    return n_bytes / 1024.0


def plot_requested_vs_allocated(rows, out_path):
    import matplotlib

    matplotlib.use("Agg")
    import matplotlib.pyplot as plt

    allocs = [r for r in rows if r["event"] == "alloc"]
    if not allocs:
        print("no allocation rows found, skipping requested_vs_allocated chart")
        return

    sections = sorted(set(r["section"] for r in allocs))
    fig, axes = plt.subplots(1, len(sections), figsize=(6 * len(sections), 5), squeeze=False)
    axes = axes[0]

    for ax, section in zip(axes, sections):
        section_rows = [r for r in allocs if r["section"] == section]
        labels = [r["label"] for r in section_rows]
        requested = [to_kb(r["requested_bytes"]) for r in section_rows]
        allocated = [to_kb(r["allocated_bytes"]) for r in section_rows]

        x = range(len(labels))
        width = 0.35
        ax.bar([i - width / 2 for i in x], requested, width, label="requested (KB)", color="#5B8FF9")
        ax.bar([i + width / 2 for i in x], allocated, width, label="actually allocated (KB)", color="#61DDAA")

        ax.set_xticks(list(x))
        ax.set_xticklabels(labels, rotation=30, ha="right")
        ax.set_ylabel("Kilobytes (KB)")
        ax.set_yscale("log")
        ax.set_title(f"{section}: requested vs. allocated memory")
        ax.legend()
        ax.grid(axis="y", linestyle="--", alpha=0.4)

    fig.suptitle("How much memory did we ask for vs. how much did the allocator actually give us?")
    fig.tight_layout()
    fig.savefig(out_path, dpi=150)
    print(f"wrote {out_path}")


def plot_freed_vs_held(rows, out_path):
    import matplotlib

    matplotlib.use("Agg")
    import matplotlib.pyplot as plt

    sections = sorted(set(r["section"] for r in rows))
    fig, axes = plt.subplots(len(sections), 1, figsize=(9, 4.5 * len(sections)), squeeze=False)
    axes = axes[:, 0]

    for ax, section in zip(axes, sections):
        section_rows = [r for r in rows if r["section"] == section]
        if not section_rows:
            continue

        steps = []
        cumulative_freed_kb = []
        returned_to_os_kb = []

        running_freed = 0
        peak_rss_kb = section_rows[0]["rss_kb"]

        for r in section_rows:
            peak_rss_kb = max(peak_rss_kb, r["rss_kb"])
            if r["event"] == "free":
                running_freed += r["requested_bytes"]

            released_kb = max(0, peak_rss_kb - r["rss_kb"])

            steps.append(r["step"])
            cumulative_freed_kb.append(to_kb(running_freed))
            returned_to_os_kb.append(released_kb)

        ax.plot(steps, cumulative_freed_kb, marker="o", label="memory we asked to free (cumulative)", color="#F6903D")
        ax.plot(steps, returned_to_os_kb, marker="o", label="memory actually returned to the OS", color="#61DDAA")
        ax.fill_between(
            steps,
            returned_to_os_kb,
            cumulative_freed_kb,
            where=[c >= r_ for c, r_ in zip(cumulative_freed_kb, returned_to_os_kb)],
            color="#F6903D",
            alpha=0.15,
            label="held by the allocator, not yet given back",
        )

        ax.set_xlabel("step (time moves left to right)")
        ax.set_ylabel("Kilobytes (KB)")
        ax.set_title(f"{section}: freed vs. actually released back to the OS")
        ax.legend(loc="upper left")
        ax.grid(linestyle="--", alpha=0.4)

    fig.suptitle("free() does not mean the memory instantly goes back to the free pool")
    fig.tight_layout()
    fig.savefig(out_path, dpi=150)
    print(f"wrote {out_path}")


def main():
    csv_path = sys.argv[1] if len(sys.argv) > 1 else "memory_log.csv"
    out_dir = sys.argv[2] if len(sys.argv) > 2 else "."

    if not os.path.exists(csv_path):
        print(f"error: could not find {csv_path}")
        print("Run `go run ./cmd/malloc-demo` first -- it writes this file automatically.")
        sys.exit(1)

    os.makedirs(out_dir, exist_ok=True)
    rows = load_rows(csv_path)

    plot_requested_vs_allocated(rows, os.path.join(out_dir, "requested_vs_allocated.png"))
    plot_freed_vs_held(rows, os.path.join(out_dir, "freed_vs_held.png"))


if __name__ == "__main__":
    main()
