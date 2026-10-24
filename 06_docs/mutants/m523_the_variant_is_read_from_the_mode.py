import pathlib
# 0.19.0 W10.2 (A-83): the variant is read from run.txt's mode field.
# Killed by TestTheHourlyRunIsJudgedOnItsDownloadsAlone.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = '\treturn f[2]\n'
assert s.count(old) == 1, 'm523'
p.write_text(s.replace(old, '\treturn f[1]\n', 1))
