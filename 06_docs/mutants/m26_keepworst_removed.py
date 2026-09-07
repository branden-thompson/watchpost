import pathlib
p = pathlib.Path("domains/globalfeed/stack.go"); s = p.read_text()
old = "\t\tkeepWorst(held)\n"
assert old in s, "m26"
p.write_text(s.replace(old, ""))
