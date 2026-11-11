import pathlib
# 0.19.5 D-8: Setup decides "the same place" by snapshot.PlaceID.
# Killed by TestSetupKeepsTwoPlacesThatShareCoordinatesButNotAZip.
p = pathlib.Path('app/setup.go'); s = p.read_text()
old = 'snapshot.PlaceID(snapshot.LocationRef{Zip: l.Zip, Lat: l.Lat, Lon: l.Lon}) == same'
assert s.count(old) == 1, 'm669'
p.write_text(s.replace(old, '(same != "" && l.Zip == loc.Zip && l.Zip != "") || (l.Lat == loc.Lat && l.Lon == loc.Lon)', 1))
