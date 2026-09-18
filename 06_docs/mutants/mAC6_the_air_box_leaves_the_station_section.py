import pathlib
# The LIVE NOW / RELAY BED box goes back to standing on its own below the station
# section, so what is on the air and whether the station is on the air are two
# regions again (D-107, HUM LEAD: "it should all be in 1 section").
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """	body := append(append([]string{""}, b.stationLine()...), b.airBox()...)"""
assert old in s, "mAC6"
p.write_text(s.replace(old, """	body := append([]string{""}, b.stationLine()...)""", 1))
