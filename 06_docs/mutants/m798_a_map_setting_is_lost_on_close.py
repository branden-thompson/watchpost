import pathlib
# 0.19.5 Q9.3 (JM-8): a saved Setting's change marks the window for saving, or closing Settings loses it.
# Killed by TestAnArrowOnASavedSettingMarksItForSaving.
p = pathlib.Path('modes/tty/map_prefs.go'); s = p.read_text()
old = '\td.mapQuakeFeed = cycleIn(quakeFeeds, itself, d.mapQuakeFeed, forward)\n\treturn d.uiTouched()\n'
assert s.count(old) == 1, 'm798'
p.write_text(s.replace(old, '\td.mapQuakeFeed = cycleIn(quakeFeeds, itself, d.mapQuakeFeed, forward)\n\treturn d\n', 1))
