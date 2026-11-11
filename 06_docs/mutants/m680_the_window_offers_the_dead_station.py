import pathlib
# 0.19.5 Q2.1 (RN-1): the relay-fault window never offers the station that went silent.
# Killed by TestASilentRelayOffersTheOtherStationsAndAPickLeads.
p = pathlib.Path('app/radio.go'); s = p.read_text()
old = '\tseen := map[string]bool{dead.Callsign: true}\n'
assert s.count(old) == 1, 'm680'
p.write_text(s.replace(old, '\tseen := map[string]bool{dead.Callsign + "x": true}\n', 1))
