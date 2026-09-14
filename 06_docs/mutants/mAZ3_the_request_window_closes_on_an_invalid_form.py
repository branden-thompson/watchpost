import pathlib
# `enter` closes the window whether or not the form can be scheduled. The
# operator believes they have scheduled a report and nothing was ever sent —
# FR-3.3 exactly: "an action must never be shown as taken unless the schedule
# took it."
p = pathlib.Path("modes/tty/request.go"); s = p.read_text()
old = """	if !d.request.valid() || d.cfg.RequestCard == nil {
		return d, nil
	}"""
assert old in s, "mAZ3"
p.write_text(s.replace(old, """	if d.cfg.RequestCard == nil {
		return d, nil
	}""", 1))
