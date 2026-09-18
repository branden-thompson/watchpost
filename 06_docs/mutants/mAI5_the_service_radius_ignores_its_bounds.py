import pathlib
# The service radius writes whatever was typed, so a stray keystroke stores a
# one-mile station or a thousand-mile one — and the storage's own clamp then
# silently disagrees with the number the window shows (D-115).
#
# Re-pointed at D-124, which moved the bounds out of this package: the guard is
# now `d.inServiceRange`, handed the floor and ceiling through `Config`. Same
# rule, same detector, new spelling.
p = pathlib.Path("modes/tty/setup.go"); s = p.read_text()
old = """		d.inServiceRange)"""
assert old in s, "mAI5"
p.write_text(s.replace(old, "		nil)", 1))
