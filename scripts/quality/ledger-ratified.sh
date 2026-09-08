#!/bin/sh
# P10 ledger gate (FR-7.3): no exemption may name a gate it is waiting for
# without carrying a ratification.
#
# WHY THIS IS NOT "no row may say 'ratify at gate'". The obvious check is a grep
# for that phrase, and it would fail on 33 of the 127 rows today — every one of
# which IS ratified and merely keeps the wording it was written with. A gate
# that fires on 33 rows nobody needs to act on is a gate that gets disabled.
#
# The obligation is the pairing: a row may say where it will be presented, or it
# may record that it WAS, but a row that says the first and never the second is
# an exemption in force that nobody approved. F-45 found 23 of those by
# ENUMERATING the file after a build log asserted the ledger was fully ratified
# — the log was true of the 15 rows in its own table and false of the file.
#
# LOCAL GATE, like `make p10`: the ledger is outside the public tree
# (.gitignore'd, red-team R2-2), so CI has no file to read. --self-test carries
# its own fixture and therefore runs anywhere, which is what lets gate-controls
# prove this still fires without needing the ledger.
set -eu

case "${1:-}" in
  ""|--self-test) ;;
  *) echo "ledger-ratified: unknown argument '$1'" >&2; exit 2 ;;
esac

# check reads a ledger file and prints the rows that are waiting and unratified.
check() {
  awk '
    /^- file: / {
      if (block != "") rows[++n] = block
      block = $0; next
    }
    { if (block != "") block = block "\n" $0 }
    END {
      if (block != "") rows[++n] = block
      for (i = 1; i <= n; i++) {
        r = rows[i]
        waiting = (r ~ /[Rr]atify at/ || r ~ /presented for ratification/)
        # "RATIFIED", "ratified by", "HUM LEAD approved" — the three forms the
        # ledger actually uses. Matched case-insensitively via tolower().
        lower = tolower(r)
        done = (lower ~ /ratified by/ || lower ~ /hum lead approved/ || lower ~ /^.*ratified [0-9]/ || lower ~ /ratified \(/ || lower ~ /"ratified"/ || lower ~ /ratified 20/)
        if (waiting && !done) {
          split(rows[i], L, "\n")
          print "  " L[1]
        }
      }
    }
  ' "$1"
}

if [ "${1:-}" = "--self-test" ]; then
  tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
  # TWO controls, because the two ways to get this wrong are opposites: missing
  # the unratified row, and flagging a ratified one whose wording is stale.
  cat > "$tmp/bad.yml" <<'EOF'
exemptions:
- file: some/pkg
  symbol: package
  rule_id: P10-05-INVARIANT-DENSITY
  reason: 'Pure helpers. Ratify at the B1a gate.'
EOF
  cat > "$tmp/good.yml" <<'EOF'
exemptions:
- file: some/pkg
  symbol: package
  rule_id: P10-05-INVARIANT-DENSITY
  reason: 'Pure helpers. Ratify at the B1a gate. RATIFIED by the HUM LEAD 2026-09-06.'
EOF
  if [ -z "$(check "$tmp/bad.yml")" ]; then
    echo "ledger-ratified SELF-TEST FAILED: an unratified waiting row was not detected"; exit 1
  fi
  if [ -n "$(check "$tmp/good.yml")" ]; then
    echo "ledger-ratified SELF-TEST FAILED: a RATIFIED row was flagged for its stale wording — the gate would fire on 33 rows nobody can act on"; exit 1
  fi
  echo "ledger-ratified self-test: both controls fired (OK)"
  exit 0
fi

LEDGER=${LEDGER:-.a2dh-p10-exemptions.yml}
if [ ! -f "$LEDGER" ]; then
  echo "ledger-ratified: no ledger at $LEDGER — this is a LOCAL gate; set LEDGER=/path/to/ledger" >&2
  exit 1
fi
bad=$(check "$LEDGER")
if [ -n "$bad" ]; then
  echo "ledger-ratified: exemption(s) in force that name a gate and carry no ratification:"
  echo "$bad"
  echo "Present them to the HUM LEAD and record the ratification IN THE ROW — never self-approve (F-45)."
  exit 1
fi
echo "ledger-ratified: OK ($(grep -c '^- file: ' "$LEDGER") rows, 0 waiting without a ratification)"
