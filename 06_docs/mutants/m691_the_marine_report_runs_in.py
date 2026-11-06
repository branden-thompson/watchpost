import pathlib
# 0.19.5 Q2.4 (RN-5): the marine report opens after two seconds of air.
# Killed by TestComposeOrdersMarineFireAndSeismic.
p = pathlib.Path('domains/radio/synth/compose.go'); s = p.read_text()
old = '\tif marineSegs := c.MarineSegments(loc.Label, reports.Maritime, imperial, now); len(marineSegs) > 0 {\n\t\tpauseLast(segs, reportPause)\n'
assert s.count(old) == 1, 'm691'
p.write_text(s.replace(old, '\tif marineSegs := c.MarineSegments(loc.Label, reports.Maritime, imperial, now); len(marineSegs) > 0 {\n', 1))
