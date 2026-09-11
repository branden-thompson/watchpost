import pathlib
# A read that runs past the window stops saying so. Five lines of a twelve-line
# report then read as the whole of a short one — the difference between "that is
# all it says" and "that is all it fits", on a station.
p = pathlib.Path("modes/tty/broadcaster_slots.go"); s = p.read_text()
old = "\t\tif i == bcReadLines-1 && len(wrapped) > bcReadLines {\n\t\t\tline = more(o, line, room)\n\t\t}\n"
assert old in s, "mN8"
p.write_text(s.replace(old, "", 1))
