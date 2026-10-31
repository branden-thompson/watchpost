import pathlib
# 0.19.0 FR-9.2: the record of a process that is gone is let go.
# Killed by TestAnUnrelatedSayIsNeverEnded.
p = pathlib.Path('domains/radio/synth/children.go'); s = p.read_text()
old = '\t\t\t_ = os.Remove(path) // gone, or a record that cannot be read\n'
assert old in s, 'm374'
p.write_text(s.replace(old, ''))
