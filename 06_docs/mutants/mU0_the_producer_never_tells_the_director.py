import pathlib
# The alert rail is wired to nothing: the ticker produces arrivals and no one
# receives them, so NO HAZARD IS EVER READ. Silent, and invisible to every test
# that builds its own schedule.
p = pathlib.Path("app/schedule.go"); s = p.read_text()
old = "\ttick.emit = s.carry\n"
new = "\t_ = s.carry\n"
assert old in s, "mU0"
p.write_text(s.replace(old, new, 1))
