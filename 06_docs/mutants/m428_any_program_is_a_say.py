import pathlib
# 0.19.0 I-4: only the say program itself is ended, not any process naming its files.
# Killed by TestAnUnrelatedSayIsNeverEnded.
p = pathlib.Path('domains/radio/synth/children.go'); s = p.read_text()
old = ' && filepath.Base(program) == "say" && '
assert old in s, 'm428'
p.write_text(s.replace(old, ' && program != "" && '))
