import pathlib
# A Finished is sent for words that were never finished: the read was cut short
# and the schedule is told a read happened that did not.
#
# RE-ANCHORED AT T3.10b. A cut-short read used to come home with NOTHING, and
# this mutant made it come home Finished. Nothing was the other half of the same
# defect — a card that says nothing stays ON AIR for ever with the band holding a
# callout for a read that has stopped (DR-24) — so it comes home FAILED now, and
# the mutation is the same one it always was.
#
# RE-ANCHORED AGAIN AT THE BUILD-EXIT RED TEAM (I-2), for the Routed field.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = '\t\treturn []lineup.Event{lineup.Failed{ID: v.ID, Reason: "the read ended before the words did", Routed: true}}'
new = '\t\treturn []lineup.Event{lineup.Finished{ID: v.ID}}'
assert old in s, "mC0"
p.write_text(s.replace(old, new, 1))
