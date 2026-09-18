import pathlib
# The bed's list stops being fenced to the station's reach, so the resolver's
# whole country-wide ranking is offered — "Lone Pine's 'Fresno' relay is
# completely inappropriate for being the relay" (D-77, D-117).
p = pathlib.Path("app/bedrelay.go"); s = p.read_text()
old = """		if st.KM*0.621371 <= radiusMi {
			kept = append(kept, st)
		}"""
assert old in s, "mAK2"
p.write_text(s.replace(old, """		kept = append(kept, st)""", 1))
