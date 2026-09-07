import pathlib
# The window boundary tightened by one instant. A disaster exactly at 24 hours
# is demoted where the ruling says it is still within the window.
p = pathlib.Path("platform/lineup/plan.go"); s = p.read_text()
old = "	if now.Sub(at) <= DisasterFreshWindow {"
new = "	if now.Sub(at) < DisasterFreshWindow {"
assert old in s, "m74"
p.write_text(s.replace(old, new, 1))
