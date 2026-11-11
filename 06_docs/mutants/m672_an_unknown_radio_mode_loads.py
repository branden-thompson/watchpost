import pathlib
# 0.19.5 D-9: an unknown [radio] mode is refused at load.
# Killed by TestAnUnknownRadioModeIsRefusedNamingTheKey.
p = pathlib.Path('platform/config/config.go'); s = p.read_text()
old = '\tif !radioModes[r.Mode] {\n'
assert s.count(old) == 1, 'm672'
p.write_text(s.replace(old, '\tif r.Mode == "never" {\n', 1))
