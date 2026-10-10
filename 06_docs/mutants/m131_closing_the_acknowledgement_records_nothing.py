import pathlib
# W2.5 (FR-11.2, A-9): closing it records the wording's version.
# Killed by TestClosingTheAcknowledgementRecordsItSeen.
p = pathlib.Path("modes/tty/map_prop_ack.go"); s = p.read_text()
old = "\td.cfg.PropagationAck = propAckVersion\n\td, update := d.askPropagation()"
assert old in s, "m131"
p.write_text(s.replace(old, "\td, update := d.askPropagation()"))
