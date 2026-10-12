import pathlib
# 0.19.0 go-tuiMaps P4 (D-174): the night and its edge are handed in with the field.
# Killed by TestTheNightIsDrawnOnThePropagationMap.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\treturn []tuimaps.Overlay{field, propNight(d.propNightMoment())}\n'
assert old in s, 'm198'
p.write_text(s.replace(old, '\treturn []tuimaps.Overlay{field}\n'))
