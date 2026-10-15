import pathlib
# 0.19.0 FR-4.1 (D-168): GIRO's badge is lit only for a snapshot its stations fed.
# Killed by TestTheStationsAreCreditedWhenTheyFedIt.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\t\t{"GIRO", shown && snap.Inputs.Stations > 0},'
assert old in s, 'm220'
p.write_text(s.replace(old, '\t\t{"GIRO", shown},'))
