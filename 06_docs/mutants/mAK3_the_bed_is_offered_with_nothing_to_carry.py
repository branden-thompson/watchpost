import pathlib
# The bed is offered whatever the station can reach, so an operator on a station
# no directory streams to can cut the programme to silence — the one outcome this
# control must not have (D-117).
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """func (b Broadcaster) bedAvailable() bool { return !b.bedTold || b.bed.Relays > 0 }"""
assert old in s, "mAK3"
p.write_text(s.replace(old, """func (b Broadcaster) bedAvailable() bool { return true }""", 1))
