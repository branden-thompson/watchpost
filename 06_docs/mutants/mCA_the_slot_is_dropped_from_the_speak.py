# RE-ANCHORED AT T3.10b: the effect list is written over three lines now that
# CueTicker carries a Slot too. The rule is unchanged.
import pathlib
# BD-8 undone for the words: the speak names no slot, so the executor reads every
# card as a location report and declines it.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "\t\tSpeak{ID: onAir.ID, Slot: onAir.Slot, Script: onAir.Script},\n"
new = "\t\tSpeak{ID: onAir.ID, Script: onAir.Script},\n"
assert old in s, "mCA"
p.write_text(s.replace(old, new, 1))
