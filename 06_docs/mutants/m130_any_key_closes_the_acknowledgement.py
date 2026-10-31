import pathlib
# W2.5 (FR-11.2, D-81): only Enter or Esc closes it.
# Killed by TestOnlyEnterOrEscCloseIt.
p = pathlib.Path("modes/tty/map_prop_ack.go"); s = p.read_text()
old = "\t\tif act, ok := d.keys.Lookup(key.String()); ok {\n\t\t\treturn d.handleModalNav(act), nil // scrolling only: it is no way out\n\t\t}\n\t\treturn d, nil"
assert old in s, "m130"
p.write_text(s.replace(old, "\t\tif act, ok := d.keys.Lookup(key.String()); ok {\n\t\t\treturn d.handleModalNav(act), nil // scrolling only: it is no way out\n\t\t}"))
