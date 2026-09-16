import pathlib
# F-110, HUM LEAD 2026-09-16: "valid alerts need to be read, expired alerts must
# never be." The composer goes back to declining the WHOLE burst when any one of
# its refs has lapsed, so a live hazard is silenced by an expired one beside it —
# and the card is re-offered and re-declined every cycle, held by the Director
# and refused here, with `heldNotice` counting it the whole time.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = """		if !e.Until.IsZero() && now.After(e.Until) {
			continue
		}"""
new = """		if !e.Until.IsZero() && now.After(e.Until) {
			return nil, false
		}"""
assert old in s, "mXP3"
p.write_text(s.replace(old, new, 1))
