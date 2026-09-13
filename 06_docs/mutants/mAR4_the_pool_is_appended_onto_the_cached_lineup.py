import pathlib
# The frame joins the two spans with `append(sched.lines, ...)` again. Both spans
# may be the memo's own copies, so this writes the pool's rows into the cached
# running order's spare capacity — and every hit afterwards replays the
# corruption. It appears one frame LATE and only at some pool sizes, which is the
# worst shape a bug can have.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """	joined := make([]string, 0, len(sched.lines)+len(pool.lines))
	joined = append(joined, sched.lines...)
	joined = append(joined, pool.lines...)
	out = append(out, b.chromeAt(joined, from, pool.off, pool.total)...)"""
assert old in s, "mAR4"
p.write_text(s.replace(old, "	out = append(out, b.chromeAt(append(sched.lines, pool.lines...), from, pool.off, pool.total)...)", 1))
