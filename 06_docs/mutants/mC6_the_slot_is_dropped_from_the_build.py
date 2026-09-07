# RE-ANCHORED at T3.10a (the build gained Refs) and again at T4.3 (it gained
# Divert, the figure the divert notice speaks). The rule about the SLOT is
# unchanged.
import pathlib
# BD-8 undone: the build no longer says what kind of card it is for, so every card
# reads as the zero slot — a location report — and every alert is declined.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """\t\treturn d, []Effect{BuildCard{ID: standby.ID, Slot: standby.Slot, Subject: standby.Subject,
\t\t\tRefs: standby.Refs, Divert: standby.Divert}}"""
new = """\t\treturn d, []Effect{BuildCard{ID: standby.ID, Subject: standby.Subject,
\t\t\tRefs: standby.Refs, Divert: standby.Divert}}"""
assert old in s, "mC6"
p.write_text(s.replace(old, new, 1))
