# With no voice the card holds the air for nothing, so the band's callout is blitted
# past before anyone can read it (P4 F10).
# RE-ANCHORED AT T3.8's completion: the rule MOVED to the Reader. The fixed hold
# that keeps a silent card readable is one rule for both callers now — the live
# takeover and the Director's Speak read through the same function — so it is
# guarded where it lives rather than in one of the two places it used to.
import pathlib
p = pathlib.Path("app/read_script.go"); s = p.read_text()
old = "\thold, start := breakingHold, time.Now()\n"
new = "\thold, start := time.Duration(0), time.Now()\n"
assert old in s, "mC8"
p.write_text(s.replace(old, new, 1))
