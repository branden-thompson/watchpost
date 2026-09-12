import pathlib
# Observer's own Tune reaches the engine while the console holds the air, so the
# operator's listening re-tunes the station's output. The lower-case `tune` stays
# open on purpose for the Director; guarding the EXPORTED method is the whole
# distinction the survey found (D-91).
p = pathlib.Path("app/radio.go"); s = p.read_text()
old = """	if !d.monitorHasTheAir() {
		return
	}
	d.tune(ref)"""
new = "\td.tune(ref)"
assert old in s, "mZ4"
p.write_text(s.replace(old, new, 1))
