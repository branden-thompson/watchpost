import pathlib
# 0.19.0 FR-9.2: a process whose start time is not the one recorded is not the one recorded.
# Killed by TestAnUnrelatedSayIsNeverEnded.
p = pathlib.Path('domains/radio/synth/children.go'); s = p.read_text()
old = 'gap <= startSlack && gap >= -startSlack'
assert old in s, 'm372'
p.write_text(s.replace(old, 'gap == gap'))
