# SUPERSEDES mA5_the_stopped_programme_is_still_built, DELETED AT D-84.
# Same inversion: it guarded `prepareNext`'s refusal to build on standby, which is now the
# defect rather than the rule.
import pathlib
# The Composer refuses to work until the station is on the air, so the card at the
# head of the line-up has no words when the operator presses SHIFT+ENTER — and
# they hear 1.03 s of network before the station says anything.
#
# HUM LEAD: "when the operator does — the line **should be ready to go** at that
# point."
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "\t\tstandby, err := next.To(Standby)\n"
new = "\t\tif !d.advances(MainTrack) {\n\t\t\treturn d, nil\n\t\t}\n\t\tstandby, err := next.To(Standby)\n"
assert old in s, "mP1"
p.write_text(s.replace(old, new, 1))
