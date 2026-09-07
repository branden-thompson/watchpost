import pathlib
# The containment stops being recorded. A broken executor is silent, and a fault
# that fails nothing leaves no sign it happened at all (DR-23).
p = pathlib.Path("app/pump.go"); s = p.read_text()
old = "		p.onFault(f, r)\n"
assert old in s, "mB7"
p.write_text(s.replace(old, "", 1))
