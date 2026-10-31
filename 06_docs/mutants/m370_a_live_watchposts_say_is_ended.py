import pathlib
# 0.19.0 FR-9.2: a say whose watchpost still runs is never ended.
# Killed by TestAnUnrelatedSayIsNeverEnded.
p = pathlib.Path('domains/radio/synth/children.go'); s = p.read_text()
old = 'return p.UID == uid && p.PPID == 1 &&'
assert old in s, 'm370'
p.write_text(s.replace(old, 'return p.UID == uid &&'))
