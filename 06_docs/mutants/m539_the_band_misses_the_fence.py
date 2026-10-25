import pathlib
# 0.19.0 RP-6: the latitude band is narrower than the fence, so places inside it are never read.
# Killed by TestAWorldScanAnswersAsTheTableDoes.
p = pathlib.Path('domains/locations/geodata/near.go'); s = p.read_text()
old = 'reach := radiusMi/milesPerDegree + 0.01 //'
assert s.count(old) == 1, 'm539'
p.write_text(s.replace(old, 'reach := radiusMi/milesPerDegree/2 //', 1))
