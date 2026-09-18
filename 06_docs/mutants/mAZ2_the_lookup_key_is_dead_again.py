import pathlib
# `[l] Lookup Location from Pool` stops working. Drawn since D-102 and bound at
# R4b; a painted control that does nothing is worse than an absent one, because
# the operator concludes the feature is broken.
#
# RE-POINTED AT THE UAT FIX, 2026-09-14. `[l]` was first built as "open the
# details of the row the pointer is on" — a misreading of the label — and the HUM
# LEAD wanted the location SEARCH box: "the <l> press is in Broadcaster UI
# EXPECTING the location search modal, not the line-up location details modal."
# It is Observer's own action now, forwarded rather than reimplemented, so the
# mutation is removing the BINDING rather than a handler.
#
# WRITTEN AGAINST THE FORMATTED SOURCE — gofmt aligns the keymap's values into a
# column, so the anchor carries that padding. mAG1 learned this the same way.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """		actLookup:        {Keys: []string{"l"}, Help: "Lookup Location"},\n"""
assert old in s, "mAZ2"
p.write_text(s.replace(old, "", 1))
