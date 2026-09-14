import pathlib
# The running order goes back to fifteen cards, which draws POSITIONS 2 to 14 —
# one row short of the reference, a slot the mock has and the operator was
# promised (D-119, HUM LEAD: "I did mean 15 all slots in the scheduled line
# should be changeable").
#
# Re-pointed at R3: the number moved to `lineup.MainTrackCap`, because "the last
# card falls off" made it a SCHEDULE rule and the console held the only copy of a
# bound the domain had to enforce. Same rule, one package along — and the mutant
# now sits where the number does, so the console reading it is covered too.
p = pathlib.Path("platform/lineup/operator.go"); s = p.read_text()
old = """const MainTrackCap = 16"""
assert old in s, "mAM2"
p.write_text(s.replace(old, """const MainTrackCap = 15""", 1))
