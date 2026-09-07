import pathlib
# The staleness check stops exempting cards that were never built.
#
# A structural card's words are fixed at proposal, so its BuiltAt is zero and
# d.now.Sub(zero) is enormous — every one of them reads as stale forever. The
# card that matters is the stale NOTICE itself: it would be found stale, raise a
# second notice, and that one a third. The loop is closed by the model (zero
# BuiltAt) rather than by a special case, which is exactly the kind of guard a
# later reader deletes as redundant.
p = pathlib.Path("platform/lineup/stale.go"); s = p.read_text()
old = "if c.State != Standby || c.BuiltAt.IsZero() {"
new = "if c.State != Standby {"
assert old in s, "mS1"
p.write_text(s.replace(old, new, 1))
