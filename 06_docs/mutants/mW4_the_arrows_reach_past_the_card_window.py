import pathlib
# modalCard drops out of handleNav's list of scrolling windows, so the window draws
# a scroll rail whose arrow keys do nothing to it and instead walk the TABLE
# underneath — the D-58 defect ("the window on top owns the keys") in the window
# that was added without being added to the list. Everything below the first twelve
# rows becomes unreachable at 80x24, which is what FR-5 measures.
p = pathlib.Path("modes/tty/nav.go"); s = p.read_text()
old = "	case modalHelp, modalDetails, modalAlerts, modalStatus, modalAbout, modalCard:"
new = "	case modalHelp, modalDetails, modalAlerts, modalStatus, modalAbout:"
assert old in s, "mW4"
p.write_text(s.replace(old, new, 1))
