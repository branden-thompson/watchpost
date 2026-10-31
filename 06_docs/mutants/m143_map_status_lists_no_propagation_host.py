import pathlib
# 0.19.0 FR-4.3: MAP STATUS lists GIRO and NOAA SWPC, as the acknowledgement says.
# Killed by TestMapStatusListsEveryPropagationHost.
p = pathlib.Path("app/maps.go"); s = p.read_text()
old = "\tout = append(out, propagationHosts()...)"
assert old in s, "m143"
p.write_text(s.replace(old, "\t_ = propagationHosts"))
