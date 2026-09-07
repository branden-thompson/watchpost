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
p = pathlib.Path("app/compose_takeover.go"); s = p.read_text()
old = "cast.Classify(worstOf(fresh).Type)"
new = "cast.Classify(fresh[0].Type)"
assert old in s, "mG1"
p.write_text(s.replace(old, new, 1))
