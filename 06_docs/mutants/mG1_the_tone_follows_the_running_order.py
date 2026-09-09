import pathlib
# The tone is taken from the FIRST card of the burst instead of its worst
# hazard. That was correct only while the burst was sorted by severity; T3.1's
# ladder orders by rung, so a fresh quake (rung 2) leads a burst whose worst
# item is a tornado warning (rung 3) and the listener gets the disaster chime
# for the wrong hazard. MVS-D-12 and MVS-D-65 both state the rule as "the most
# serious event sets the tone" — the first card was a mechanism that happened
# to satisfy it, not the rule.
# RE-ANCHORED AT T3.7: the rule MOVED. Choosing the tone is content, so it went
# to the Composer with the rest of the card's words (MVS-D-77). Following the
# anchor to its new home rather than deleting the mutant is the difference
# between a rule that moved and a rule that was lost — which the corpus guard
# has now caught five times in this release.
# RE-ANCHORED AT 0.15.0 B1 (#18). The call moved behind toneClassOfEvent, the
# one owner of "which sound does this event make" — cast.Classify reads a
# product's words, which cannot carry the civil-emergency family. A pure move
# for this mutant: the rule it guards, that the burst's WORST hazard sets the
# tone, is unchanged and still lives on this line.
p = pathlib.Path("app/compose_takeover.go"); s = p.read_text()
old = "toneClassOfEvent(worstOf(fresh))"
new = "toneClassOfEvent(fresh[0])"
assert old in s, "mG1"
p.write_text(s.replace(old, new, 1))
