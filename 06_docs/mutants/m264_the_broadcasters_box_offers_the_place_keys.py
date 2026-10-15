import pathlib
# 0.19.0 FR-2.4, D-195: the Broadcaster's controls box says the red ring.
# Killed by TestTheBroadcastersMapOffersNoPlaceKeys.
p = pathlib.Path('modes/tty/map_boxes.go'); s = p.read_text()
old = '\tif !d.mapRole().placeKeys {\n\t\treturn " red: service"'
assert old in s, 'm264'
p.write_text(s.replace(old, '\tif false {\n\t\treturn " red: service"'))
