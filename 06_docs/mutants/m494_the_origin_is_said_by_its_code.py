import pathlib
# 0.19.0 D-231: the origin abroad is said by its country's name in the words.
# Killed by TestAPlaceAbroadIsSaidByItsCountry.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '	origin.name = geo.SpokenPlace(origin.name, origin.country) // a place abroad by its country (D-231)\n'
assert old in s, 'm494'
p.write_text(s.replace(old, '', 1))
