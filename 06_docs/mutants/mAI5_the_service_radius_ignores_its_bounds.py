import pathlib
# The service radius writes whatever was typed, so a stray keystroke stores a
# one-mile station or a thousand-mile one — and the storage's own clamp then
# silently disagrees with the number the window shows (D-115).
p = pathlib.Path("modes/tty/setup.go"); s = p.read_text()
old = """		func(v int) bool { return v >= serviceRadiusMin && v <= serviceRadiusMax })"""
assert old in s, "mAI5"
p.write_text(s.replace(old, "		nil)", 1))
