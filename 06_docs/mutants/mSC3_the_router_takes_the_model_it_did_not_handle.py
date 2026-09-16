import pathlib
# D-159. `update` returns the model `keyAction` handed back even when the key was
# NOT answered, instead of keeping its own. Today the two are equal — no
# fall-through path mutates the Router — so this mutant is a TRIPWIRE of the D-42
# shape: it survives while that property holds, and the day a case starts
# mutating before it falls through, the surface below silently receives a Router
# that has already been half-changed by a rule that declined to act.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """			if m, cmd, handled := r.keyAction(msg, k, a); handled {
				return m, cmd
			}"""
new = """			m, cmd, handled := r.keyAction(msg, k, a)
			if handled {
				return m, cmd
			}
			if rr, ok := m.(Router); ok {
				r = rr
			}"""
assert old in s, "mSC3"
p.write_text(s.replace(old, new, 1))
