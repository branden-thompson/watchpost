import pathlib
# W2.5 (FR-11.2, A-9): closing it records the wording's version.
# Killed by TestClosingTheAcknowledgementRecordsItSeen.
p = pathlib.Path("modes/tty/map_prop_ack.go"); s = p.read_text()
old = "\td.cfg.PropagationAck = propAckVersion\n\tsave := d.cfg.SavePropagationAck"
assert old in s, "m131"
p.write_text(s.replace(old, "\tsave := d.cfg.SavePropagationAck\n\tif save != nil {\n\t\treturn d, nil\n\t}"))
