import pathlib
# 0.19.0 FR-4.2, D-81: nothing is fetched before the acknowledgement closes.
# Killed by TestNothingIsFetchedBeforeTheAcknowledgement.
p = pathlib.Path("modes/tty/map_prop.go"); s = p.read_text()
old = "\t\treturn d.openPropAck(), cmd // before anything"
assert old in s, "m140"
p.write_text(s.replace(old, "\t\td.modal = modalMap\n\t\td, up := d.askPropagation()\n\t\treturn d.openPropAck(), tea.Batch(cmd, up) // before anything"))
