#!/bin/sh
# p10-unmatched_test.sh — the scope helper's own tests (0.14.0 P1 Task 1.0).
#
# Three fixture repositories, each a complete little history, exercise the two
# scoping rules the helper gets wrong without them:
#
#   1. a docs-only diff       -> every ledger row dormant, exit 0
#   2. a rename, symbol same  -> dormant, exit 0
#   3. a rename, symbol changed, nothing matched -> unmatched, exit 1
#
# POSIX sh only (the script under test runs under dash on Linux and bash 3.2 on
# macOS, and so does this).
#
# usage: scripts/quality/p10-unmatched_test.sh

set -eu
here=$(cd "$(dirname "$0")" && pwd)
script="$here/p10-unmatched.sh"
[ -x "$script" ] || { echo "p10-unmatched_test: $script is not executable"; exit 1; }

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM
fails=0

# newrepo <dir> — a git repo with one Go file and one doc, committed.
newrepo() {
  mkdir -p "$1" && cd "$1"
  git init -q .
  git config user.email t@example.invalid
  git config user.name  Test
  git config commit.gpgsign false
  cat > narrate.go <<'GO'
package fixture

// Attention is the symbol the ledger rows below are keyed on.
func Attention() int {
	return 1
}

func Untouched() int {
	return 2
}
GO
  echo "# doc" > README.md
  git add -A && git commit -qm base
}

# ledger <file> <symbol> — a one-row ledger.
ledger() {
  cat > .a2dh-p10-exemptions.yml <<YAML
- file: $1
  symbol: $2
  rule_id: P10-04-FUNCTION-SIZE
  reason: fixture
YAML
}

# report <base> — a p10.json naming base and carrying NO exempted findings, so
# every ledger row is unmatched and only the scoping decides the exit code.
report() {
  cat > p10.json <<JSON
{
  "base": "$1",
  "findings": null
}
JSON
}

# check <name> <expected-exit> — run the helper here and compare.
check() {
  set +e
  out=$("$script" p10.json .a2dh-p10-exemptions.yml 2>&1)
  got=$?
  set -e
  if [ "$got" != "$2" ]; then
    echo "FAIL $1: exit $got, want $2"
    printf '%s\n' "$out" | sed 's/^/      /'
    fails=$((fails + 1))
  else
    echo "ok   $1 (exit $got)"
  fi
}

# --- 1. a docs-only diff: P10 governs compiled code, so nothing this run could
#        find applies to any row. Every row is dormant and the gate passes.
newrepo "$tmp/docs"
base=$(git rev-parse HEAD)
echo "more docs" >> README.md
git commit -qam docs
ledger narrate.go Attention
report "$base"
check "docs-only diff leaves every row dormant" 0

# --- 2. a rename with the symbol unchanged: the row is re-keyed to the new
#        path, which looks brand-new against the base. It is not: the pair's
#        diff has no hunk in Attention, so the row is dormant.
newrepo "$tmp/rename-same"
base=$(git rev-parse HEAD)
git mv narrate.go director.go
git commit -qm rename
ledger director.go Attention
report "$base"
check "rename with the symbol unchanged is dormant" 0

# --- 3. a rename with the symbol changed: the pair's diff has a hunk inside
#        Attention, so the row IS in scope — and having matched no finding it
#        is dead and must fail the gate.
newrepo "$tmp/rename-changed"
base=$(git rev-parse HEAD)
git mv narrate.go director.go
cat > director.go <<'GO'
package fixture

// Attention is the symbol the ledger rows below are keyed on.
func Attention() int {
	x := 1
	x += 41 // the body changed: this row is in scope again
	return x
}

func Untouched() int {
	return 2
}
GO
git commit -qam rename-and-change
ledger director.go Attention
report "$base"
check "rename with the symbol changed is unmatched" 1

# --- 4. a control: with no base resolved the helper keeps its pre-FULL-GIT
#        behaviour and treats everything as in scope, so the same dead row
#        still fails. This is what distinguishes case 1 from an empty file
#        list caused by an unresolvable base.
newrepo "$tmp/nobase"
echo "more docs" >> README.md
git commit -qam docs
ledger narrate.go Attention
report "0000000000000000000000000000000000000000"
check "an unresolvable base keeps everything in scope" 1

cd "$here"
if [ "$fails" != 0 ]; then
  echo "p10-unmatched_test: $fails case(s) failed"
  exit 1
fi
echo "p10-unmatched_test: all cases passed"
