import pathlib
# F-D1 undone: prepareNext refuses a card that arrives with its own words, the
# way the old invariant did. The card never leaves ADMITTED, Next offers it for
# ever, and every card queued behind it goes unread — DR-3 failing from the
# other side.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "\tstandby, err := next.To(Standby)\n"
new = "\tif !next.Script.Empty() {\n\t\treturn d, nil\n\t}\n\tstandby, err := next.To(Standby)\n"
assert old in s, "mD2"
p.write_text(s.replace(old, new, 1))
