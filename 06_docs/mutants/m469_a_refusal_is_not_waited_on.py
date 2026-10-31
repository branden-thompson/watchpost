import pathlib
# 0.19.0 W10.3 (D-208, D-221): wspr.live's refusal is waited on, 60 s doubling.
# Killed by TestTheFetchBacksOffAndKeepsOnlyTheRateHeaders.
p = pathlib.Path('tools/reachscore/spots.go'); s = p.read_text()
old = '		f.sleep(wait)'
assert old in s, 'm469'
p.write_text(s.replace(old, '		_ = wait', 1))
