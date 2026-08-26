#!/bin/sh
# p10-unmatched.sh — ledger entries that matched no finding (plan §1 "0
# unmatched ledger entries"; red-team R2-11, SC-1). An exemption that
# matches nothing is dead ceremony at best and a silenced future finding
# at worst; the Q1 gate deletes them.
#
# usage: scripts/quality/p10-unmatched.sh <p10.json> [ledger=.a2dh-p10-exemptions.yml]
# exit 0 when every entry matched; 1 with the unmatched entries listed.
#
# Matching mirrors the tool: an entry's file is exact for a symbol entry
# and a directory prefix for a package entry; the rule id must be equal;
# only findings the tool marked exempted count.

set -eu
json=${1:?p10.json}; ledger=${2:-.a2dh-p10-exemptions.yml}
[ -r "$json" ] || { echo "p10-unmatched: cannot read $json"; exit 1; }
[ -r "$ledger" ] || { echo "p10-unmatched: cannot read $ledger"; exit 1; }

# Exempted findings as "file rule" lines (the JSON is pretty-printed: one key per line).
findings=$(awk -F'"' '
  /"file":/    { file=$4 }
  /"rule_id":/ { rule=$4 }
  /"exempted": true/ { print file, rule }' "$json")

# Ledger entries as "file symbol rule" lines.
entries=$(awk '
  /^ *- file:/ { sub(/^ *- file: */, ""); file=$0; sym=""; rule="" }
  /^ *symbol:/ { sub(/^ *symbol: */, ""); sym=$0 }
  /^ *rule_id:/ { sub(/^ *rule_id: */, ""); rule=$0; print file, sym, rule }' "$ledger")

bad=0
os=$(uname -s)
printf '%s\n' "$entries" | while read -r file sym rule; do
  [ -n "$file" ] || continue
  # A platform-tagged file is invisible to a run on another OS (red-team SC-3): check it there, not here.
  case "$file" in
    *_windows.go) [ "${os#MINGW}" != "$os" ] || [ "${os#MSYS}" != "$os" ] || { echo "skipped (checked on windows): $file $rule"; continue; } ;;
    *_unix.go|*_linux.go|*_darwin.go) [ "$os" != "Darwin" ] && [ "$os" != "Linux" ] && { echo "skipped (checked on unix): $file $rule"; continue; } ;;
  esac
  if [ "$sym" = "package" ] && [ "$rule" = "P10-05-INVARIANT-DENSITY" ]; then
    # the tool matches a directory to a density finding by prefix — for P10-05 only (red-team SC-1)
    hit=$(printf '%s\n' "$findings" | awk -v f="$file" -v d="$file/" -v r="$rule" '($1==f || index($1, d)==1) && $2==r {print; exit}')
  else
    hit=$(printf '%s\n' "$findings" | awk -v f="$file" -v r="$rule" '$1==f && $2==r {print; exit}')
  fi
  [ -n "$hit" ] || { echo "unmatched: $file ($sym) $rule"; bad=1; }
  echo "$bad" > "${TMPDIR:-/tmp}/p10-unmatched.$$"
done
result=$(cat "${TMPDIR:-/tmp}/p10-unmatched.$$" 2>/dev/null || echo 0); rm -f "${TMPDIR:-/tmp}/p10-unmatched.$$"
if [ "$result" = "1" ]; then echo "p10-unmatched: ledger entries matched nothing (delete or re-key them)"; exit 1; fi
echo "p10-unmatched: every ledger entry matched a finding"
