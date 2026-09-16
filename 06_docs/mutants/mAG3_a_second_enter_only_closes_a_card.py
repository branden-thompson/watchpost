import pathlib
# D-113. A second `enter` closes the CARD window alone, so on a POOL row — where
# `enter` now opens Details — the key falls through to Observer again, one window
# along from the defect this rule was written for.
#
# Re-pointed 2026-09-16 (D-159): the keymap switch moved out of `update` into
# `keyAction` when `update` was split at cyclomatic 43. One tab shallower; the
# rule and the mutation are unchanged.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """\t\tif r.active == SurfaceBroadcaster &&
\t\t\t(r.observer.modal == modalCard || r.observer.modal == modalDetails) {"""
new = """\t\tif r.active == SurfaceBroadcaster && r.observer.modal == modalCard {"""
assert old in s, "mAG3"
p.write_text(s.replace(old, new, 1))
