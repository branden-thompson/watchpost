import pathlib
# The LIVE slot goes back to saying nothing on standby — the state the HUM LEAD
# called out at D-84 ("we'll need to design an empty-state for that slot") and
# designed at D-89. An operator opening the console meets an unexplained empty row
# where the programme goes, with no way to tell "nothing is scheduled" from "the
# render is broken".
#
# RE-ANCHORED AT D-110. D-89's grey BOX retired when LIVE became one row of the
# air box (D-95); the SENTENCE is what survived, and this is where it is said.
p = pathlib.Path("modes/tty/broadcaster_air.go"); s = p.read_text()
old = "\t\treturn bcCardInset + bcStandbyNotice\n"
new = "\t\treturn \"\"\n"
assert old in s, "mX1"
p.write_text(s.replace(old, new, 1))
