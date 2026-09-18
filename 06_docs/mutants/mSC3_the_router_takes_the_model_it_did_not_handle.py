import pathlib
# D-159. `routeKey` takes the model `keyAction` handed back even when the key was
# NOT answered, instead of keeping its own. Today the two are equal — no
# fall-through path in that switch mutates the Router — so this mutant is a
# TRIPWIRE of the D-42 shape: it SURVIVES BY DESIGN while that property holds,
# and fails the day a case starts mutating before it falls through, which is
# exactly when the surface below would silently receive a Router already
# half-changed by a rule that declined to act.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """			if m, cmd, handled := r.keyAction(msg, k, a); handled {
				return m, cmd, true
			}"""
new = """			m, cmd, handled := r.keyAction(msg, k, a)
			if handled {
				return m, cmd, true
			}
			if rr, ok := m.(Router); ok {
				r = rr
			}"""
assert old in s, "mSC3"
p.write_text(s.replace(old, new, 1))
