import pathlib
# The running order goes back to fifteen cards, which draws POSITIONS 2 to 14 —
# one row short of the reference, a slot the mock has and the operator was
# promised (D-119, HUM LEAD: "I did mean 15 all slots in the scheduled line
# should be changeable").
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """const MainTrackSlots = 16"""
assert old in s, "mAM2"
p.write_text(s.replace(old, """const MainTrackSlots = 15""", 1))
