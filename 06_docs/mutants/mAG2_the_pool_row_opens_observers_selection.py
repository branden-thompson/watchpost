import pathlib
# `enter` on a pool row goes back to falling through to Observer, which opens the
# details for the row OBSERVER has selected — so the operator points at Rancho
# Penasquitos and is shown Oceanside (D-113).
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """	if at := r.broadcaster.poolSelection(); at >= 0 {
		pool := r.broadcaster.area.Pool
		if at >= len(pool) {
			return r, false // the pointer is past the pool; nothing to open
		}
		r.observer = r.observer.showLocation(pool[at])
		return r, true
	}"""
assert old in s, "mAG2"
p.write_text(s.replace(old, "", 1))
