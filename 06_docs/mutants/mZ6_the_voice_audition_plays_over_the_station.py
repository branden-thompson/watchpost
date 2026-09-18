import pathlib
# A voice preview from Settings auditions over the output while the console holds
# the air — and "ON AIR never means the antenna is radiating": Watchpost produces
# audio a human patches into a transmitter, so the sample goes out with the
# programme. That is "sneaking under to get on the air" exactly (D-91).
p = pathlib.Path("app/voices.go"); s = p.read_text()
old = """	if !d.monitorHasTheAir() {
		d.voiceNote("preview is unavailable while the station holds the air")
		return
	}
"""
assert old in s, "mZ6"
p.write_text(s.replace(old, "", 1))
