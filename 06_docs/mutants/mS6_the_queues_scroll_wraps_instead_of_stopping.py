import pathlib
# RE-AIMED 2026-09-12 (D-101): the queue's scroll became the POINTER's walk.  The
# rule is unchanged — it CLAMPS at the ends and never wraps, because a list that
# jumped from its last row to its first would lose the operator's place, and on a
# running order 'where am I' is the question the numbers exist to answer.
# Scrolling above the top wraps to a negative offset instead of stopping, so the
# window jumps somewhere the operator did not ask for. A running order is
# addressed by NUMBER, and "where am I" is the question those numbers exist to
# answer.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = '\tb.selected = max(0, min(n-1, b.selected+by))'
new = '\tb.selected = (b.selected + by + n) % n'
assert old in s, "mS6"
p.write_text(s.replace(old, new, 1))
