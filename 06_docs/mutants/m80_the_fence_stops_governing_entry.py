import pathlib
# The fence dropped from the planner — it becomes an ordering input only, which
# is what the design says it is NOT: it governs entry (DR-13).
p = pathlib.Path("platform/lineup/plan.go"); s = p.read_text()
old = """		if !s.Fence.Admits(a) {
			continue
		}
"""
assert old in s, "m80"
p.write_text(s.replace(old, "", 1))
