import pathlib
# The pool windows its band and column titles along with its rows, so the operator
# who scrolls down loses the names of the columns they are reading (D-106, HUM
# LEAD: "so they dont disappear when I scroll down").
p = pathlib.Path("modes/tty/broadcaster_pool.go"); s = p.read_text()
old = """	headN := len(table) - len(rows)"""
assert old in s, "mAC2"
p.write_text(s.replace(old, """	headN := 0""", 1))
