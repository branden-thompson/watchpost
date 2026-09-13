import pathlib
# The bed is offered whatever the station can reach, so an operator on a station
# no directory streams to can cut the programme to silence — the one outcome this
# control must not have (D-117).
#
# Re-pointed at D-125, which moved the relay count off `BedMsg` and onto its own
# message: same rule, same detector, and the predicate now reads a field with one
# writer instead of one that three publishers could zero.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """func (b Broadcaster) bedAvailable() bool { return !b.bedRelaysTold || b.bedRelays > 0 }"""
assert old in s, "mAK3"
p.write_text(s.replace(old, """func (b Broadcaster) bedAvailable() bool { return true }""", 1))
