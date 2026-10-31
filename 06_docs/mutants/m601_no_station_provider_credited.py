import pathlib
# 0.19.0 D-41, D-340: About credits GIRO and not each station's data provider.
# Killed by TestEveryPropagationSourceIsCredited.
p = pathlib.Path('app/credits.go'); s = p.read_text()
old = '\t\t\t{Badge: "GIRO", What: "Each Station\'s Data Provider", Host: "giro.uml.edu", Note: "Acknowledged as GIRO lists them for each ionosonde"}, // D-41, D-340\n'
assert s.count(old) == 1, 'm601'
p.write_text(s.replace(old, '', 1))
