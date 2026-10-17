import pathlib
# 0.19.0 FR-9.2 (I-4): only the user's own process is ended.
# Killed by TestAnUnrelatedSayIsNeverEnded.
p = pathlib.Path('domains/radio/synth/children.go'); s = p.read_text()
old = 'return p.UID == uid && p.PPID == 1 &&'
assert old in s, 'm369'
p.write_text(s.replace(old, 'return p.PPID == 1 &&'))
