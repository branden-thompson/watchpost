import pathlib
# D-130. The radius test is dropped, so ANY place the geocoder knows answers as
# broadcastable — Lone Pine at 200 miles reads the same as Vista at four. The
# station would schedule a report about somewhere it cannot reach, which is the
# whole reason the fence exists (D-72).
p = pathlib.Path("app/pool.go"); s = p.read_text()
old = "		within := globalfeed.WithinMiles(s.transmitter.Lat, s.transmitter.Lon, ref.Lat, ref.Lon, s.radiusMi)"
# THE CALL IS KEPT, ITS ANSWER IGNORED. Deleting it orphans the globalfeed
# import and the build fails — which is INVALID, not evidence.
new = "		within := globalfeed.WithinMiles(s.transmitter.Lat, s.transmitter.Lon, ref.Lat, ref.Lon, s.radiusMi) || true"
assert old in s, "mCC2"
p.write_text(s.replace(old, new, 1))
