import pathlib
# 0.19.0 W10.2 (A-83): the hourly run is held to the default refresh's target.
# Killed by TestTheHourlyRunIsJudgedOnItsDownloadsAlone.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = '\t\twireTarget = g1WireMBHourly\n'
assert s.count(old) == 1, 'm521'
p.write_text(s.replace(old, '\t\twireTarget = g1WireMBHour\n', 1))
