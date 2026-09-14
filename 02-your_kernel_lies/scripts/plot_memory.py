#!/usr/bin/env python3
"""Create a simple visualization from fork-demo CSV output."""

import csv
import sys
from pathlib import Path

import matplotlib.pyplot as plt


def load(path):
    rows = []
    with open(path, newline="") as f:
        for r in csv.reader(f):
            if len(r) != 7 or r[0] != "FORK_STATE":
                continue
            rows.append({
                "who": r[1], "stage": r[2],
                "rss": float(r[3]), "private": float(r[4]),
                "shared": float(r[5]), "pss": float(r[6]),
            })
    return rows


def main():
    path = Path(sys.argv[1] if len(sys.argv) > 1 else "memory_log.csv")
    rows = load(path)
    if not rows:
        raise SystemExit("No FORK_STATE rows found in the log.")

    labels = [f"{r['who']}\n{r['stage'].replace('_', ' ')}" for r in rows]

    plt.figure(figsize=(11, 6))
    plt.plot(labels, [r["private"] for r in rows], marker="o", label="Private dirty")
    plt.plot(labels, [r["shared"] for r in rows], marker="o", label="Shared dirty")
    plt.plot(labels, [r["pss"] for r in rows], marker="o", label="PSS")
    plt.ylabel("Memory (KB)")
    plt.title("What changes around fork() and copy-on-write")
    plt.xticks(rotation=25, ha="right")
    plt.legend()
    plt.tight_layout()
    out = path.with_name("fork_cow_memory.png")
    plt.savefig(out, dpi=160)
    print(f"Wrote {out}")


if __name__ == "__main__":
    main()
