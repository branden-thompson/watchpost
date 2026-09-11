# SUPERSEDES m102_a_stopped_station_admits_a_read, DELETED AT D-84.
# It weakened the gate this one RESTORES: the rule ran the other way until D-84, and a mutant
# that guards a retired rule is a test of nothing.
import pathlib
# Admission is gated on the air again, which is what D-84 overturned: a station on
# standby admits no cards, so the operator has ten empty slots and nothing to
# inspect, reorder or drop until AFTER they have gone on the air.
#
# HUM LEAD, 2026-09-11: "being able to see, manage, and change the line up PRIOR
# to going on air is a fundamental requirement — otherwise the user might as well
# just use Observer."
p = pathlib.Path("platform/lineup/topoff.go"); s = p.read_text()
old = "func (d Director) onOffered(ev Offered) (Director, []Effect) {\n"
new = "func (d Director) onOffered(ev Offered) (Director, []Effect) {\n\tif !d.advances(MainTrack) {\n\t\treturn d, nil\n\t}\n"
assert old in s, "mP0"
p.write_text(s.replace(old, new, 1))
