#!/bin/sh
# dump.sh — ask a running watchpost for a diagnostic dump and print the
# directory it wrote (plan §2.1, Q0 task 6).
#
# usage: scripts/quality/dump.sh <pid>
#
# Prefers /debug/dump on the opt-in loopback server (works everywhere,
# returns the path); falls back to SIGUSR1 (Unix) and prints the newest
# profiles directory. Dumps are rate-bound to one a minute by the app.

set -eu
pid=${1:?pid}
if dir=$(curl -fs --max-time 30 http://127.0.0.1:6060/debug/dump 2>/dev/null); then
  echo "$dir"; exit 0
fi
kill -USR1 "$pid"
sleep 3
case "$(uname -s)" in
  Darwin) root="$HOME/Library/Caches/watchpost/profiles" ;;
  *)      root="${XDG_CACHE_HOME:-$HOME/.cache}/watchpost/profiles" ;;
esac
ls -1d "$root"/*/ 2>/dev/null | sort | tail -1
