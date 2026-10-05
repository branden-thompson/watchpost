import pathlib
# modalCard is declared without its own nav, so the window draws a scroll rail
# whose arrow keys do nothing to it and instead walk the TABLE underneath - the
# D-58 defect ("the window on top owns the keys") in a window added without
# its scroll. Everything below the first twelve rows becomes unreachable at
# 80x24, which is what FR-5 measures. Re-pointed at F-184's declarations
# (window_keys.go), where the scrolling windows are now named; before, it was
# handleNav's hand-written list.
p = pathlib.Path("modes/tty/window_keys.go"); s = p.read_text()
old = "	case modalHelp, modalDetails, modalAlerts, modalStatus, modalAbout, modalCard:"
new = "	case modalCard:\n\t\treturn windowKeys{claim: claimActions}, true\n	case modalHelp, modalDetails, modalAlerts, modalStatus, modalAbout:"
assert old in s, "mW4"
p.write_text(s.replace(old, new, 1))
