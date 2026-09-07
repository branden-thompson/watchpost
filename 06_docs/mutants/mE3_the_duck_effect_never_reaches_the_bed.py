import pathlib
# The Duck effect is accepted and does nothing, so the rail reads over a
# broadcast at full volume — the effect the Director emits around a whole drain
# (MVS-D-67) becomes a no-op nobody notices.
#
# RE-ANCHORED: the Duck effect HOLDS the bed now rather than only dipping it,
# because a per-sequence take-back would otherwise lift it between cards.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = "\t\tx.mc.hold()\n"
assert old in s, "mE3"
p.write_text(s.replace(old, "", 1))
