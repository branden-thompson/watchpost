import pathlib
# 0.19.0 FR-9.2: only a say with a watchpost-say- path in its arguments is ended.
# Killed by TestAnUnrelatedSayIsNeverEnded.
p = pathlib.Path('domains/radio/synth/children.go'); s = p.read_text()
old = '&& strings.Contains(p.Args, sayMarker) &&'
assert old in s, 'm371'
p.write_text(s.replace(old, '&&'))
