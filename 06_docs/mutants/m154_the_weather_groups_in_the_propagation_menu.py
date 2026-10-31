import pathlib
# D-159: in the Propagation mode the menu is the tints and MAP DETAILS.
# Killed by TestTheMenuInThePropagationModeShowsTintsAndDetail.
p = pathlib.Path('modes/tty/map_boxes.go'); s = p.read_text()
old = "\tif d.mapMode() == modePropagation { // D-159: the tints and the map's detail"
assert old in s, 'm154'
p.write_text(s.replace(old, "\tif false { // D-159: the tints and the map's detail"))
