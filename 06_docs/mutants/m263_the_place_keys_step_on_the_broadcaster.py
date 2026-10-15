import pathlib
# 0.19.0 FR-2.4, D-195: [ and ] do not move the Broadcaster's map.
# Killed by TestTheBroadcastersMapOffersNoPlaceKeys.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\t\tif d.mapRole().placeKeys { // D-195'
assert old in s, 'm263'
p.write_text(s.replace(old, '\t\tif true { // D-195'))
