import pathlib
# The card-less exemption in `contained` made unreachable: a publish that comes
# apart now emits Failed{ID: ""}, garbage on the wire whose harmlessness depends
# entirely on the Director's lookup finding nothing under an empty id.
#
# ANCHORED ON THE LINE BELOW IT, not on `if !ofCard {` alone — that text also
# appears in byCard, where an earlier version of this mutant landed by mistake
# and reported SURVIVED against a rule it was not testing.
p = pathlib.Path("app/pump.go"); s = p.read_text()
old = """		if !ofCard {
			out = nil // nothing to fail: the schedule is unharmed, the record stands"""
new = """		if !ofCard && id == "\\x00never" {
			out = nil // nothing to fail: the schedule is unharmed, the record stands"""
assert old in s, "mB5"
p.write_text(s.replace(old, new, 1))
