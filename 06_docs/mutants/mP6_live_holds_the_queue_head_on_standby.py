import pathlib
# The LIVE slot draws the head of the line-up on a station that is not on the air,
# so the operator is shown a card as LIVE while the station is silent — and the
# card they were told goes out FIRST is the one sitting in the slot labelled "what
# is playing now".
#
# HUM LEAD: "LIVE should remain EMPTY … The UP NEXT card … will be the first thing
# that goes ON AIR when the human operator hits SHIFT+ENTER."
#
# Re-pointed 2026-09-16 (D-156): `i -= b.liveOffset()` became
# `i = b.indexForSlot(i)` when the arithmetic got one owner at its third caller.
p = pathlib.Path("modes/tty/broadcaster_slots.go"); s = p.read_text()
old = "\ti = b.indexForSlot(i)\n"
assert old in s, "mP6"
p.write_text(s.replace(old, "", 1))
