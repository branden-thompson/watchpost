import pathlib
# 0.19.0 I-4: a record a day old whose process runs under another parent is let go.
# Killed by TestAnOrphanThatCannotBeEndedIsTriedAgain.
p = pathlib.Path('domains/radio/synth/children.go'); s = p.read_text()
old = '\t\tcase p.PPID != 1 && now.Sub(started) < recordMaxAge:\n'
assert old in s, 'm429'
p.write_text(s.replace(old, '\t\tcase p.PPID != 1:\n'))
