import pathlib
# D-170: every provider serves a feature and has a credited badge.
# Killed by TestEveryStatusSourceIsGroupedAndCredited.
p = pathlib.Path('app/endpoints.go'); s = p.read_text()
old = '\t\t"usgs":       {Feature: tty.FeatureAdverse, Badge: "USGS"},\n'
assert old in s, 'm187'
p.write_text(s.replace(old, ''))
