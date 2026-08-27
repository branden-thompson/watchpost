# ADR-01 — Diagnostic dump trigger (C1)

**Status:** accepted (PLAN, 2026-08-26; built in Q0) · **Owner:** HUM LEAD

## Context
The problem statement is about a process that has run for days. Profiling hooks that must be chosen
at launch (`WATCHPOST_DEBUG_PPROF=1`) cannot attribute the process a person already has open (LR-5).

## Options
1. `SIGUSR1` → write a dump set under the cache dir (Unix); the env hook serves Windows.
2. A key in the `[S]` modal that toggles the pprof server — needs the UI focused; useless headless.
3. A trigger file polled every minute — a timer the app runs forever.
4. Drop the signal; `soak.sh` curls the existing loopback server (red-team CQ-7).

## Decision
Option 1 plus the env hook, with `/debug/dump` and `/debug/counters` on the loopback server so every
platform has a trigger. Declined 4 because OQ-2 is about a process *not* launched with the hook.

## Consequences
One `//go:build` pair (`app/dump_unix.go`, `app/dump_windows.go`) — two budgeted P10-08 entries. Bounds:
one dump in flight, ≥ 60 s apart, twelve kept, 0700/0600, never `debug.WriteHeapDump`. Zero idle cost.
