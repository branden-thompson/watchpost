import pathlib
# `enter` closes the window whether or not the form can be scheduled. The
# operator believes they have scheduled a report and nothing was ever sent —
# FR-3.3 exactly: "an action must never be shown as taken unless the schedule
# took it."
#
# Re-pointed 2026-09-16 (D-157): the nil-hook guard moved to the TOP of
# `requestSchedule` when the window learned the four-way `onSubmit` answer, so
# the validity gate now stands alone. The mutation overrides it rather than
# deleting it — dropping the block outright would dereference a nil ref and
# panic, which is a CRASH and not the silent false confirmation this measures.
p = pathlib.Path("modes/tty/request.go"); s = p.read_text()
old = """	if !d.request.valid() {
		return d, nil
	}
	ref, at, kinds :="""
new = """	if false {
		return d, nil
	}
	ref, at, kinds :="""
assert old in s, "mAZ3"
p.write_text(s.replace(old, new, 1))
