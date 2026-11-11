import pathlib
# 0.19.5 Q6.4 (A-16): the prefs a window saves are the Settings in effect.
# Killed by TestThePrefsSavedAreTheSettingsInEffect.
p = pathlib.Path('modes/tty/setup_ui.go'); s = p.read_text()
old = '\t\tMapQuakeFeed:      d.mapQuakeFeed,\n'
assert s.count(old) == 1, 'm782'
p.write_text(s.replace(old, '\t\tMapQuakeFeed:      quakeFeedDefault,\n', 1))
