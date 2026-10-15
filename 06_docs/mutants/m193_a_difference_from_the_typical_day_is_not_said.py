import pathlib
# 0.19.0 D-135: where now differs from a typical day, the chart says so.
# Killed by TestADisagreementWithTheTypicalDayIsSaid.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\tif propDiffers(answers[row].Bands, c.key.hour) {\n'
assert old in s, 'm193'
p.write_text(s.replace(old, '\tif false {\n'))
