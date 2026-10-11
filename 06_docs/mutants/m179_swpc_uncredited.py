import pathlib
# FR-4.1: every source the propagation library names is in About's credits.
# Killed by TestEveryPropagationSourceIsCredited.
p = pathlib.Path('app/credits.go'); s = p.read_text()
old = '\t\t\t{Badge: "SWPC", What: "Ionosphere (GloTEC) & Solar Indices", Host: "services.swpc.noaa.gov"},\n'
assert old in s, 'm179'
p.write_text(s.replace(old, ''))
