import pathlib
# 0.19.0 D-204: on the Observer the map does not move to a place found.
# Killed by TestAFoundPlaceIsAnsweredAndNotKept.
p = pathlib.Path('modes/tty/map_prop_find.go'); s = p.read_text()
old = '\tif m := d.mapPane.m; m != nil && d.propRole().target {'
assert old in s, 'm441'
p.write_text(s.replace(old, '\tif m := d.mapPane.m; m != nil {'))
