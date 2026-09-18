import pathlib
# The app stops sending the service radius' bounds, so the window is never told
# what they are. It then refuses every radius the operator types — fail-closed,
# which is the RIGHT direction, and still a broken setting. The wiring is the
# only thing holding this together now that the window has no constants of its
# own, which is exactly the trade D-124 made.
p = pathlib.Path("app/dashboard.go"); s = p.read_text()
old = """		ServiceRadiusMinMi: config.MinServiceRadiusMi,
		ServiceRadiusMaxMi: config.MaxServiceRadiusMi,
"""
assert old in s, "mAS1"
p.write_text(s.replace(old, "", 1))
