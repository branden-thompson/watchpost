import pathlib
# Scrolling above the top wraps to a negative offset instead of stopping, so the
# window jumps somewhere the operator did not ask for. A running order is
# addressed by NUMBER, and "where am I" is the question those numbers exist to
# answer.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\tb.queueOff = max(0, b.queueOff+by)"
new = "\tb.queueOff = b.queueOff + by"
assert old in s, "mS6"
p.write_text(s.replace(old, new, 1))
