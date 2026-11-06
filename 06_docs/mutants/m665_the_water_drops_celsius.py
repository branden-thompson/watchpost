import pathlib
# 0.19.5 D-6 (DEF-3): the water temperature takes the conditions' wording.
# Killed by TestTheWaterTemperatureSaysTheListenersUnit.
p = pathlib.Path('domains/radio/synth/marine.go'); s = p.read_text()
old = '\treturn degrees(*c, imperial)\n'
assert s.count(old) == 1, 'm665'
p.write_text(s.replace(old, '\tif imperial {\n\t\treturn degrees(*c, imperial)\n\t}\n\treturn fmt.Sprintf("%.0f degrees", *c)\n', 1))
