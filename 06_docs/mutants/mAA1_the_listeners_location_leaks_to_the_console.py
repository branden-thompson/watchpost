import pathlib
# The listener's DEFAULT LOCATION becomes a shared setting, so it appears in the
# console's Settings window. D-18 row 1 rules it Observer's, and D-72 split the
# station's epicentre out precisely because they are two facts with two owners —
# an operator editing "default location" from the console is editing the wrong one.
p = pathlib.Path("modes/tty/setup_rows.go"); s = p.read_text()
old = "rowLocation, groupData, scopeObserver"
new = "rowLocation, groupData, scopeShared"
assert old in s, "mAA1"
p.write_text(s.replace(old, new, 1))
