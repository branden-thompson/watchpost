import pathlib
# Tab walks to the next group without asking whether this surface draws it, so on
# the console it lands in ALERTS - EVENTS or RELAY REPLAY — groups with no rows on
# screen. The focus is then somewhere the operator cannot see and the keyboard
# reads as dead (D-92).
#
# Re-pointed 2026-09-25 (0.18.0 D-62): tab now walks TABS, and the same defect is
# a tab's first row chosen without asking the surface - the console would land
# on the Maps tab, and Observer on the Broadcaster tab, whose rows it never draws.
p = pathlib.Path("modes/tty/setup_tabs.go"); s = p.read_text()
old = "if tabOfGroup(setupTable()[id].group) == t && d.shownOnSurface(id) {"
new = "if tabOfGroup(setupTable()[id].group) == t {"
assert old in s, "mAA4"
p.write_text(s.replace(old, new, 1))
