import pathlib
# The hold is never cleared, so the broadcast never comes back after a rail
# drain — worse than pumping, because nothing recovers it.
#
# Re-pointed 2026-09-15 (D-147): `unhold` writes through `setUnder`, the helper
# the dupes gate asked for when `tickerDeck.setScope` became its second caller.
# The rule is unchanged — the mutation still sets the hold instead of clearing
# it — and it now reaches the value rather than the assignment.
p = pathlib.Path("app/mastercontrol.go"); s = p.read_text()
old = "\tsetUnder(&m.mu, &m.held, false)\n"
assert old in s, "mE5"
p.write_text(s.replace(old, "\tsetUnder(&m.mu, &m.held, true)\n", 1))
