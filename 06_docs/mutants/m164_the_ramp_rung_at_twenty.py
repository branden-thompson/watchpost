import pathlib
# D-166: the ramp's third rung ends at 15 minutes.
# Killed by TestTheFreshnessRampHasSevenRungs.
p = pathlib.Path('platform/render/freshness.go'); s = p.read_text()
old = '\tcase age < 15*time.Minute:\n\t\treturn Fresh15'
assert old in s, 'm164'
p.write_text(s.replace(old, '\tcase age < 20*time.Minute:\n\t\treturn Fresh15'))
