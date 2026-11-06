import pathlib
# 0.19.5 Q2.8 (RN-13): the second swell has a row of its own.
# Killed by TestTheMarineSectionDrawsEverySwellAndTheBuoysWind.
p = pathlib.Path('modes/tty/detail_marine.go'); s = p.read_text()
old = '\tif m.SecondarySwellHeight != nil {\n'
assert s.count(old) == 1, 'm705'
p.write_text(s.replace(old, '\tif false {\n', 1))
