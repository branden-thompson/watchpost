#!/usr/bin/env bash
# lint.sh — golangci-lint (which runs staticcheck) as a BASELINE + RATCHET.
#
# WHY NOT A CLEAN BILL. Neither linter has ever run as a gate here, so the tree
# starts with findings that predate the gate; demanding zero would mean either a
# large unrelated change or a gate nobody turns on. The ratchet is what a gate
# is FOR: this tree's known findings are recorded once, and the build fails on
# anything that is not one of them.
#
# IT RATCHETS BOTH WAYS. A finding that has been fixed also fails, with the line
# to delete — otherwise the baseline silently keeps room for a defect to come
# back into, which is the stale-exemption shape platform/closedset exists to
# stop.
#
# THE FINGERPRINT IS FILE + RULE, NOT FILE + LINE. A line number moves whenever
# anything above it does, and a baseline that churns on every edit is one people
# regenerate without reading.
set -uo pipefail
cd "$(dirname "$0")/.."

BASELINE=scripts/lint-baseline.txt
tmp=$(mktemp) || exit 1
trap 'rm -f "$tmp" "$tmp.now"' EXIT

# THE JSON GOES TO A FILE. Both writers default to stdout, so asking for JSON
# there yields the text report interleaved with it — five lines of "JSON" that
# no parser accepts.
golangci-lint run --output.json.path "$tmp" ./... >/dev/null 2>&1 || true
if [ ! -s "$tmp" ]; then
  echo "lint: golangci-lint produced nothing — the gate cannot report on a run that did not happen" >&2
  exit 1
fi

# file<TAB>linter/code — the code is staticcheck's when it names one.
python3 - "$tmp" >"$tmp.now" <<'PY'
import json, re, sys
with open(sys.argv[1]) as f:
    doc = json.load(f)
seen = {}
for i in doc.get("Issues") or []:
    path = i.get("Pos", {}).get("Filename", "?")
    rule = i.get("FromLinter", "?")
    m = re.match(r"([A-Z]{2}\d{4}):", i.get("Text", ""))
    if m:
        rule += "/" + m.group(1)
    key = f"{path}\t{rule}"
    seen[key] = seen.get(key, 0) + 1
for k in sorted(seen):
    print(f"{seen[k]}\t{k}")
PY

if [ "${1-}" = "--update" ]; then
  cp "$tmp.now" "$BASELINE"
  echo "lint: baseline written ($(wc -l <"$BASELINE" | tr -d ' ') entries)"
  exit 0
fi

if [ ! -f "$BASELINE" ]; then
  echo "lint: no baseline yet — run 'make lint-update' and read what it records" >&2
  exit 1
fi

new=$(comm -13 <(sort "$BASELINE") <(sort "$tmp.now"))
gone=$(comm -23 <(sort "$BASELINE") <(sort "$tmp.now"))
rc=0
if [ -n "$new" ]; then
  echo "lint: NEW findings — this change introduced them:" >&2
  echo "$new" | sed 's/^/  /' >&2
  rc=1
fi
if [ -n "$gone" ]; then
  echo "lint: findings in the baseline that no longer occur. Take them out with 'make lint-update'," >&2
  echo "      or the baseline keeps room for them to come back:" >&2
  echo "$gone" | sed 's/^/  /' >&2
  rc=1
fi
[ $rc -eq 0 ] && echo "lint: no new findings ($(wc -l <"$BASELINE" | tr -d ' ') baselined)"
exit $rc
