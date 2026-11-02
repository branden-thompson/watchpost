import pathlib
# 0.19.3 A-107 (11): the tab row takes its short names before it wraps.
# Killed by TestTheTabRowFitsTheNarrowWindow.
p = pathlib.Path('modes/tty/setup_tabs.go'); s = p.read_text()
old = "\tif row := d.tabRowNamed(o, tabs, setupTab.Short); render.Width(row) <= width {\n"
assert s.count(old) == 1, 'm627'
p.write_text(s.replace(old, "\tif row := d.tabRowNamed(o, tabs, setupTab.Short); render.Width(row) <= 0 {\n", 1))
