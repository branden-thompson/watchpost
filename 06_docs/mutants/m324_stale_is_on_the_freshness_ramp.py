import pathlib
# 0.19.0 FR-5.2: a stale field is said in the alert's tone.
# Killed by TestAStaleFieldIsSaidStale.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\t\ttone = render.AlertLabel // FR-5.2\n'
assert old in s, 'm324'
p.write_text(s.replace(old, '\t\ttone = render.FreshnessTone(age) // FR-5.2\n'))
