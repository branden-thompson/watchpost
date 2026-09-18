import pathlib
# The settle goes back to publishing the DIRECTOR's bed ref over the operator's
# chosen relay. A settle happens on every tick, so the console's relay row reverts
# to "(no relay tuned)" about a second after every keypress and the operator has no
# way to know which relay their station is on — reported in UAT 2026-09-11.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = """	if x.selected != nil {
		if chosen := x.selected(); chosen != "" {
			return chosen
		}
	}
"""
assert old in s, "mY1"
p.write_text(s.replace(old, "", 1))
