import pathlib
# 0.19.5 Q2.4 (RN-5): the marine report's first segment carries its manifest line.
# Killed by TestComposeOrdersMarineFireAndSeismic.
p = pathlib.Path('domains/radio/synth/compose.go'); s = p.read_text()
old = '\t\tmarineSegs[0].Source, marineSegs[0].Detail = marineSource, marineSpan(reports.Maritime, now)\n'
assert s.count(old) == 1, 'm690'
p.write_text(s.replace(old, '', 1))
