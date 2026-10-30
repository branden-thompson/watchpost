import pathlib
# 0.19.0 D-336: a listener's change with no file yet is refused as a first-run config.
# Killed by TestAChangeOnAFirstRunIsKept.
p = pathlib.Path('platform/config/config.go'); s = p.read_text()
old = '\tcfg.FirstRun = false\n\treturn Save(cfg)\n}'
assert s.count(old) == 1, 'm591'
p.write_text(s.replace(old, '\treturn Save(cfg)\n}', 1))
