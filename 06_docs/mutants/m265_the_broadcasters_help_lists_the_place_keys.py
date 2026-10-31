import pathlib
# 0.19.0 FR-2.4, D-195: Help lists only the keys the role binds.
# Killed by TestTheBroadcastersMapOffersNoPlaceKeys.
p = pathlib.Path('modes/tty/help_about.go'); s = p.read_text()
old = 'if d.mapRole().placeKeys || (act != actMapPrev && act != actMapNext) {'
assert old in s, 'm265'
p.write_text(s.replace(old, 'if true {'))
