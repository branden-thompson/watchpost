import pathlib
# The Director is never told the programme is RUNNING. It starts Stopped — a
# station comes up silent, deliberately — and Stop was the only power it heard,
# so advances(MainTrack) is false for the life of the process and the bed never
# moves on. Watchlist looks like it simply does nothing, on the relay path and
# the synth path alike.
#
# Found at UAT 2026-09-04, and no test caught it because every fixture sent
# Powered{Running} itself: the tests supplied what the wiring had forgotten.
#
# RE-ANCHORED AGAIN AT D-74: a tune reports the MONITOR's power now, because the
# station's and the operator's are two fields. The rule is unchanged — the report
# must exist, or the rotation never advances.
#
# RE-ANCHORED 2026-09-10. The start used to be a side effect of the DECK changing
# mode, which is what the defect was; it is reported from the tune now, BEFORE
# the relay/synth fork. The mutant follows the RULE, not the line — and it had
# stopped applying, so the rule was UNMEASURED.
p = pathlib.Path("app/radio.go"); s = p.read_text()
old = "\td.tell(lineup.Monitored{Running: true})\n"
new = ""
assert old in s, "mM3"
p.write_text(s.replace(old, new, 1))
