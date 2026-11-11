import pathlib
# 0.19.5 Q2.8 (RN-13): the buoy's wind is in the listener's units.
# Killed by TestTheMarineSectionDrawsEverySwellAndTheBuoysWind.
p = pathlib.Path('platform/render/units.go'); s = p.read_text()
old = '\tif o.Units == UnitC {\n\t\treturn fmt.Sprintf("%.0f km/h", units.KmhOf(*mps))\n'
assert s.count(old) == 1, 'm704'
p.write_text(s.replace(old, '\tif false {\n\t\treturn fmt.Sprintf("%.0f km/h", units.KmhOf(*mps))\n', 1))
