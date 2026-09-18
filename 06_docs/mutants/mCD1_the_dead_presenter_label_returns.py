import pathlib
# D-131. The card footer names a PRESENTER again — the label of a control that
# was removed (v3 plan item 6, "the per-card PRESENTER control"). It reports
# `N/A` on every card, because the thing that could have set it no longer
# exists: a field the operator cannot change, telling them nothing, on every row
# of the line-up (HUM LEAD, UAT 2026-09-14).
#
# THE RISK IS RECURRENCE, NOT SURVIVAL. `Card.ReadBy` is still there and still
# resolved — LIVE NOW and the detail window both say it — so putting the label
# back beside it compiles and reads plausibly. That is exactly why the absence
# is pinned.
p = pathlib.Path("modes/tty/broadcaster_slots.go"); s = p.read_text()
old = "	return bcCardInset + \" \" + o.KeyCap(handle) + \"  Read / Manage\""
new = "	return bcCardInset + \" \" + o.KeyCap(handle) + \"  Read / Manage\" + \"    PRESENTER: N/A\""
assert old in s, "mCD1"
p.write_text(s.replace(old, new, 1))
