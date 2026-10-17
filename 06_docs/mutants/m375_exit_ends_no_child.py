import pathlib
# 0.19.0 FR-9.1: EndChildren ends every say running.
# Killed by TestNoSayOutlivesAnOrderlyExit.
p = pathlib.Path('domains/radio/synth/children.go'); s = p.read_text()
old = '\t\tif cmd.Process != nil && cmd.Process.Kill() == nil {\n'
assert old in s, 'm375'
p.write_text(s.replace(old, '\t\tif cmd.Process != nil && false {\n'))
