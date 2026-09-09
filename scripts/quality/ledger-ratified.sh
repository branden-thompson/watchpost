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

# check reads a ledger file and prints the rows that carry no RATIFICATION FIELD.
#
# IT NO LONGER READS PROSE, and that is the whole of this rewrite (red team,
# 2026-09-08). The first version scored a row "waiting" if it contained
# "ratify at" and "done" if it contained any of six substrings — so a fixture of
# four exemptions, every one of them in force and unapproved, returned OK:
#
#   "Ratify at the B1a gate. NOT ratified by anyone yet."  matched `ratified by`
#   "Pending HUM LEAD sign-off at REVIEW."                 matched nothing
#   "To be approved at the B1a gate."                      matched nothing
#   "Awaiting ratification at the B1a gate."               matched nothing
#
# A gate that reads phrasing cannot be sound: it must guess every way a human
# might write approval, and it scores the NEGATION of its own pattern as a match.
#
# THE DEFAULT IS INVERTED. A row is unratified unless a STRUCTURED FIELD says
# otherwise — `ratified: <date>` — so a rewording cannot grant an exemption and
# silence is never approval. That is the only shape that survives someone
# writing the reason differently from everyone before them.
check() {
  awk '
    /^- file: / { if (block != "") rows[++n] = block; block = $0; next }
    { if (block != "") block = block "\n" $0 }
    END {
      if (block != "") rows[++n] = block
      for (i = 1; i <= n; i++) {
        if (rows[i] !~ /\n[[:space:]]*ratified:[[:space:]]*[\047\042]?[0-9]{4}-[0-9]{2}-[0-9]{2}/) {
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
  # THE BAD FIXTURE IS THE ONE THAT FOOLED THE OLD GATE. Every row here is an
  # exemption in force that nobody approved, and each is phrased the way a human
  # actually writes it — including one whose text contains the word "ratified"
  # inside its own negation.
  cat > "$tmp/bad.yml" <<'EOF'
exemptions:
- file: some/pkg
  symbol: package
  rule_id: P10-05-INVARIANT-DENSITY
  reason: 'Ratify at the B1a gate. NOT ratified by anyone yet.'
- file: other/pkg
  symbol: package
  rule_id: P10-05-INVARIANT-DENSITY
  reason: 'Pending HUM LEAD sign-off at REVIEW.'
- file: third/pkg
  symbol: package
  rule_id: P10-05-INVARIANT-DENSITY
  reason: 'Awaiting ratification at the B1a gate.'
EOF
  cat > "$tmp/good.yml" <<'EOF'
exemptions:
- file: some/pkg
  symbol: package
  rule_id: P10-05-INVARIANT-DENSITY
  reason: 'Pure helpers; the quota would assert what the return statement guarantees.'
  ratified: 2026-09-06
EOF
  if [ "$(check "$tmp/bad.yml" | wc -l | tr -d ' ')" != "3" ]; then
    echo "ledger-ratified SELF-TEST FAILED: expected all 3 unratified rows, got:"; check "$tmp/bad.yml"; exit 1
  fi
  if [ -n "$(check "$tmp/good.yml")" ]; then
    echo "ledger-ratified SELF-TEST FAILED: a row carrying a ratified: field was flagged"; exit 1
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
  echo "ledger-ratified: exemption(s) in force with no 'ratified: <date>' field:"
  echo "$bad"
  echo "Present them to the HUM LEAD and record 'ratified: <date>' IN THE ROW — never self-approve (F-45)."
  exit 1
fi
echo "ledger-ratified: OK ($(grep -c '^- file: ' "$LEDGER") rows, every one carrying a ratified: date)"
