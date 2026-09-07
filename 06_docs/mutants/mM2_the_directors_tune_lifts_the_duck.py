import pathlib
# RE-ANCHORED: the seam is named cutTo, because P10 resolves methods by NAME and
# a field called `tune` collided with radioDeck.tune, which is genuinely in a
# call cycle. The rule is unchanged.
# The Director's tune lifts the alert duck. Every tune it asks for is AUTOMATIC
# — a dwell elapsed, a cycle ended — so the next location's report comes in at
# full volume over a breaking alert that is still reading. This is the defect a
# capital letter used to carry (radioDeck.Tune vs tune), arriving from the new
# owner of the decision.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = """	x.cutTo(v.Ref)
	return nil"""
new = """	x.mc.takeBack()
	x.cutTo(v.Ref)
	return nil"""
assert old in s, "mM2"
p.write_text(s.replace(old, new, 1))
