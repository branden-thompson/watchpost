import pathlib
# D-155. EVERY becomes ANY: the card goes the moment the FIRST of its hazards
# lapses. A burst is ONE card carrying many hazards (MVS-D-77), so a flood
# advisory expiring takes a live tornado warning off the air with it — the
# conservative rule inverted, in the one direction where being wrong is a
# hazard the listener never hears.
p = pathlib.Path("platform/lineup/stale.go"); s = p.read_text()
old = """			live := false
			for _, a := range c.From { // bounded by the card (P10-02)
				if a.Until.IsZero() || !d.now.After(a.Until) {
					live = true
					break
				}
			}
			if !live {"""
new = """			live := true
			for _, a := range c.From { // bounded by the card (P10-02)
				if !a.Until.IsZero() && d.now.After(a.Until) {
					live = false
					break
				}
			}
			if !live {"""
assert old in s, "mXP2"
p.write_text(s.replace(old, new, 1))
