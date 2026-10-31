import pathlib
# 0.19.0 W10.2 (A-83): the hourly run's CPU and RSS comparison counts towards the exit.
# Killed by TestTheHourlyRunIsJudgedOnItsDownloadsAlone.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = 'counts := base.key == against && !hourly'
assert s.count(old) == 1, 'm522'
p.write_text(s.replace(old, 'counts := base.key == against', 1))
