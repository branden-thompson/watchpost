import pathlib
# 0.19.0 FR-3.7 (D-133): Help says the focus limit.
# Killed by TestTheFocusLimitIsSaid.
p = pathlib.Path('modes/tty/help_about.go'); s = p.read_text()
old = '\tlines = append(lines, d.helpLegend(o), "", " "+propFocusLimit) // FR-3.7, D-133\n'
assert old in s, 'm348'
p.write_text(s.replace(old, '\tlines = append(lines, d.helpLegend(o)) // FR-3.7, D-133\n'))
