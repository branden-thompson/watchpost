import pathlib
# 0.19.0 I-4: a process is read again just before it is ended, so a process ID reused between the two is never ended.
# Killed by TestAnOrphanThatCannotBeEndedIsTriedAgain.
p = pathlib.Path('domains/radio/synth/children.go'); s = p.read_text()
old = '\treturn alive && orphanOf(p, uid, started)\n'
assert old in s, 'm427'
p.write_text(s.replace(old, '\treturn alive || orphanOf(p, uid, started)\n'))
