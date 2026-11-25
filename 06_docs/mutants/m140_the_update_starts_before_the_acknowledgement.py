import pathlib
# 0.19.0 FR-4.2, D-81: nothing is fetched before the acknowledgement closes.
# Killed by TestNothingIsFetchedBeforeTheAcknowledgement.
p = pathlib.Path("modes/tty/map_prop.go"); s = p.read_text()
old = '\t\treturn mm, cmd, mm.openPropAck() // before anything'
assert old in s, "m140"
p.write_text(s.replace(old, '\t\tmm, up := mm.askPropagation(e)\n\t\treturn mm, tea.Batch(cmd, up), mm.openPropAck() // before anything'))
