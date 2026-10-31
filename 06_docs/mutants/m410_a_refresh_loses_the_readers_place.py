import pathlib
# 0.19.0 D-211: the sentence the reader was at is kept at the top.
# Killed by TestARefreshKeepsTheReadersPlace.
p = pathlib.Path('modes/tty/map_prop_place.go'); s = p.read_text()
old = '\t\t\td = d.keepWordsPlace(place) // the words may have moved: find the place again\n'
assert old in s, 'm410'
p.write_text(s.replace(old, ''))
