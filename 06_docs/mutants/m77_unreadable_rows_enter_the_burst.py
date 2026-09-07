import pathlib
# The rung-zero skip dropped. A forecast becomes a takeover — "an outlook is
# what MIGHT happen, and the burst is for what IS".
p = pathlib.Path("platform/lineup/plan.go"); s = p.read_text()
old = """		if rung == 0 {
			continue
		}
"""
assert old in s, "m77"
p.write_text(s.replace(old, "", 1))
