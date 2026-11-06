import pathlib
# 0.19.5 Q2.8 (RN-13): the buoy's wind row carries its gusts.
# Killed by TestTheMarineSectionDrawsEverySwellAndTheBuoysWind.
p = pathlib.Path('modes/tty/detail_marine.go'); s = p.read_text()
old = '\t\t\tgust = "(gusts " + o.Wind(m.WindGust) + ")"\n'
assert s.count(old) == 1, 'm703'
p.write_text(s.replace(old, '', 1))
