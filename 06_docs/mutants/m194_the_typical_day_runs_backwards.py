import pathlib
# 0.19.0 D-135: each hour of the typical day is drawn from its own open hours.
# Killed by TestTheDaysOpenHoursAreShown.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\t\t\tb.WriteString(d.propGlyph(propanswers.OpenAt(s.OpenHours, day.Add(time.Duration(h)*time.Hour)), false))\n'
assert old in s, 'm194'
p.write_text(s.replace(old, '\t\t\tb.WriteString(d.propGlyph(propanswers.OpenAt(s.OpenHours, day.Add(time.Duration(23-h)*time.Hour)), false))\n'))
