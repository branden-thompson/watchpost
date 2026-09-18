import pathlib
# The Director declines a request onto a full track. HUM LEAD, 2026-09-14: "The
# Director also executes the will of the Operator … For 0.16.0 that answer should
# be NO, it doesn't refuse the human operator." A refusal here is also the FR-3.3
# failure in reverse: the console would report a scheduled card the schedule
# never took.
p = pathlib.Path("platform/lineup/operator.go"); s = p.read_text()
old = """	at, ok := l.scheduleIndex(t, to)
	if err := invariant.Check(ok, "a card is inserted at a position the RUNNING ORDER has"); err != nil {
		return l, err
	}"""
new = """	at, ok := l.scheduleIndex(t, to)
	if err := invariant.Check(ok, "a card is inserted at a position the RUNNING ORDER has"); err != nil {
		return l, err
	}
	if l.visible(t) >= MainTrackCap {
		return l, errors.New("the running order is full")
	}"""
assert old in s, "mAY4"
p.write_text(s.replace(old, new, 1))
