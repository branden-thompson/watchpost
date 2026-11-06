import pathlib
# 0.19.5 Q2.1 (RN-1): a picked station's mounts are tried first.
# Killed by TestASilentRelayOffersTheOtherStationsAndAPickLeads.
p = pathlib.Path('app/radio.go'); s = p.read_text()
old = '\td.engine.Start(append(lead, rest...), callsign)\n'
assert s.count(old) == 1, 'm681'
p.write_text(s.replace(old, '\td.engine.Start(urls, callsign)\n', 1))
