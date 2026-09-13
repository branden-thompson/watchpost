import pathlib
# The memo stores the pair and then never recognises it: `ok` stays false, so
# every frame is a miss. PERFECTLY CORRECT AND COMPLETELY USELESS — the frame is
# right, the guard is happy, and the saving the memo was added for quietly never
# arrives. A cache that only ever misses is the failure no correctness test can
# see.
p = pathlib.Path("modes/tty/broadcaster_memo.go"); s = p.read_text()
old = "	m.ok, m.key, m.sched, m.pool = true, key, sched, pool"
assert old in s, "mAR6"
p.write_text(s.replace(old, "	m.ok, m.key, m.sched, m.pool = false, key, sched, pool", 1))
