import pathlib
# The lane's air becomes the STATION's air again: `airOnce` refuses a card while
# anything is reading anywhere, which is what the rule said until D-82.
#
# A tornado warning then cannot take the air while a location report holds it,
# and the operator hears the hazard when the weather finishes — a wait as long as
# one report. It was inert for as long as the main track had no reader (F-91);
# building one is what made it a defect on the safety path.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "\tif _, busy := d.lineup.OnAir(track); busy {"
new = "\tif busy := d.lineup.anyOnAir(); busy {"
assert old in s, "mN0"
p.write_text(s.replace(old, new, 1))
