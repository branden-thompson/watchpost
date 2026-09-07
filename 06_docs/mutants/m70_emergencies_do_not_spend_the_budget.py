import pathlib
# "Exempt from Max" read literally — the emergency orders lead AND leave the
# whole budget behind them, so a burst with one evacuation order reads Max+1.
p = pathlib.Path("platform/lineup/plan.go"); s = p.read_text()
old = """		out = append(out, c)
		spent++"""
new = """		out = append(out, c)
		if !emergency {
			spent++
		}"""
assert old in s, "m70"
p.write_text(s.replace(old, new, 1))
