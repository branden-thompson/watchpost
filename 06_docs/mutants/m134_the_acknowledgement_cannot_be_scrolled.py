import pathlib
# W2.5 (FR-11.4): every line of it can be brought on screen at the floor.
# Killed by TestEveryLineOfEveryWindowIsReachableAtTheFloor.
p = pathlib.Path("modes/tty/map_prop_ack.go"); s = p.read_text()
old = '\t\t\treturn mm, nil, mapWant{ModalNav: act} // scrolling only: it is no way out'
assert old in s, "m134"
p.write_text(s.replace(old, '\t\t\t_ = act\n\t\t\treturn mm, nil, mapWant{} // scrolling only: it is no way out'))
