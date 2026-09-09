import pathlib
# The busy guard is removed from Read, so a read asked for while one is already
# running starts a SECOND one: two voices over each other, two marks, and a
# restore for a broadcast that was ducked once. Toggle's press gate does not
# cover this — that gate serialises the DECISION between two presses (mK3);
# this guard is what makes the decision's outcome inert when the answer is
# "already reading".
#
# Held by TestEventReaderDucksSpeaksRestoresAndOverlaysThePanel only since the
# assertion was PINNED: before that the stubbed voice and stubbed sleep let the
# first read finish before the second was asked for, and the test passed on
# timing rather than on this guard. It passed on macOS and failed on a Linux
# runner for exactly that reason.
p = pathlib.Path("app/severe_read.go"); s = p.read_text()
old = "	r.mu.Lock()\n	if r.busy {\n		r.mu.Unlock()\n		return\n	}\n"
assert old in s, "mK8"
p.write_text(s.replace(old, "	r.mu.Lock()\n", 1))
