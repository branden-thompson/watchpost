import pathlib
# 0.19.0 FR-5.5, D-198: the map's words say each row's nearest station.
# Killed by TestEachRowShowsItsNearestStation.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\t\tif !once {\n\t\t\tsay += "; " + d.propStationWords(a.Nearest)\n'
assert old in s, 'm280'
p.write_text(s.replace(old, '\t\tif !once && false {\n\t\t\tsay += "; " + d.propStationWords(a.Nearest)\n'))
