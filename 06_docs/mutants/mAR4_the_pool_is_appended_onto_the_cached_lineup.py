import pathlib
# The frame joins the two spans with `append(sched.lines, ...)` again. Both spans
# may be the memo's own copies, so this writes the pool's rows into the cached
# running order's spare capacity — and every hit afterwards replays the
# corruption. It appears one frame LATE and only at some pool sizes, which is the
# worst shape a bug can have.
p = pathlib.Path("modes/tty/broadcaster_memo.go"); s = p.read_text()
old = """	out := make([]string, 0, len(sched.lines)+len(pool.lines))
	out = append(out, sched.lines...)
	return append(out, pool.lines...)"""
assert old in s, "mAR4"
p.write_text(s.replace(old, "	return append(sched.lines, pool.lines...)", 1))
