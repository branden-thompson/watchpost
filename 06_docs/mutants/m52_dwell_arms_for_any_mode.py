import pathlib
# The rotation advances whatever the listener's repeat mode is, so a station on
# Repeat Off or Repeat One moves itself on when its cycle ends — a rotation
# nobody asked for.
#
# RE-ANCHORED (T3.2b) to the CYCLE-END path. The deck's armDwell guard was
# `repeat != Watchlist || mode != "live"`; both halves are the Director's now and
# each has its own mutant — mK1 the live-relay half, this one the Watchlist half.
#
# IT TARGETS onEnded RATHER THAN dwellElapsed, deliberately: the same deletion in
# dwellElapsed is MASKED by the "an advance restarts the turn it just spent"
# invariant, which re-asks the predicate and is degenerate at a zero dwell — it
# suppresses the effect, so the behaviour stays right and no test can tell the
# rule went missing. A mutant that a defensive check hides is not evidence.
p = pathlib.Path("platform/lineup/bed.go"); s = p.read_text()
old = """	if d.settings.Dwell <= 0 {
		return d, nil // not Watchlist: the rotation does not move on by itself
	}
"""
assert old in s, "m52"
p.write_text(s.replace(old, "", 1))
