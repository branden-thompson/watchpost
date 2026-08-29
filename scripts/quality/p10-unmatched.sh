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
#
# Scope (0.13.0, FULL GIT): the tool scans the diff against its base
# (`base` in the JSON — merge-base with main). An entry for code OUTSIDE
# that diff cannot match anything this run and is DORMANT, not dead; only an
# entry inside the diff that matched nothing is dead. Dormant entries are
# counted, never failed — a row ratified for 0.11.0 code stays until that
# code changes again and the finding either returns or does not.

set -eu
json=${1:?p10.json}; ledger=${2:-.a2dh-p10-exemptions.yml}
[ -r "$json" ] || { echo "p10-unmatched: cannot read $json"; exit 1; }
[ -r "$ledger" ] || { echo "p10-unmatched: cannot read $ledger"; exit 1; }

# The scan's base and the files changed against it (tracked and untracked Go
# files): the scope inside which an unmatched entry is dead.
base=$(awk -F'"' '/"base":/ { print $4; exit }' "$json")
changed=""
if [ -n "$base" ] && git rev-parse -q --verify "$base^{commit}" >/dev/null 2>&1; then
  changed=$( { git diff --name-only "$base" -- '*.go'; git ls-files --others --exclude-standard -- '*.go'; } | sort -u)
fi
in_scope() { # $1 file, $2 symbol — can this entry match a finding of THIS scan?
  [ -z "$changed" ] && return 0 # no base known: everything is in scope (the pre-FULL-GIT behaviour)
  # A package row is a standing, HUM-LEAD-ratified exemption for the whole
  # package: it matches when a density finding lands in it and is dormant
  # otherwise — never dead by this check (a person retires it).
  [ "$2" = "package" ] && return 1
  # A symbol row is in scope when its file is new, or when a diff hunk of
  # that file touches the symbol (git names the enclosing function in the
  # hunk header) — the analyzers report changed functions only.
  printf '%s\n' "$changed" | awk -v f="$1" '$0==f { found=1; exit } END { exit !found }' || return 1
  git cat-file -e "$base:$1" 2>/dev/null || return 0
  git diff -U0 "$base" -- "$1" | awk -v s="$2" '/^@@/ && index($0, s "(") > 0 { found=1; exit } END { exit !found }'
}

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
dormant=0
os=$(uname -s)
: > "${TMPDIR:-/tmp}/p10-dormant.$$"
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
  if [ -z "$hit" ]; then
    if in_scope "$file" "$sym"; then echo "unmatched: $file ($sym) $rule"; bad=1; else echo d >> "${TMPDIR:-/tmp}/p10-dormant.$$"; fi
  fi
  echo "$bad" > "${TMPDIR:-/tmp}/p10-unmatched.$$"
done
result=$(cat "${TMPDIR:-/tmp}/p10-unmatched.$$" 2>/dev/null || echo 0); rm -f "${TMPDIR:-/tmp}/p10-unmatched.$$"
dormant=$(wc -l < "${TMPDIR:-/tmp}/p10-dormant.$$" | tr -d ' '); rm -f "${TMPDIR:-/tmp}/p10-dormant.$$"
if [ "$result" = "1" ]; then echo "p10-unmatched: ledger entries in scope matched nothing (delete or re-key them)"; exit 1; fi
echo "p10-unmatched: every in-scope ledger entry matched a finding ($dormant dormant, outside the diff against ${base:-<no base>})"
