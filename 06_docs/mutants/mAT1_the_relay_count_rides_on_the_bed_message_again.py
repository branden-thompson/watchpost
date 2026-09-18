import pathlib
# The relay count goes back onto `BedMsg`, where three publishers write the
# message and exactly one of them knows the count. The other two leave it at zero
# and silently retract the resolver's answer, so the console disables a bed that
# is CARRYING — D-117 firing backwards, telling the operator there is no relay
# while one is streaming.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """	case BedRelaysMsg:"""
assert old in s, "mAT1"
new = """	case BedRelaysMsg:
		_ = v
	case BedMsg2Unused:"""
# Simpler and truer to the defect: make BedMsg's arrival clear the count, which is
# exactly what storing the whole message used to do.
s = s.replace("		b.bed = v\n", "		b.bed, b.bedRelays, b.bedRelaysTold = v, 0, true\n", 1)
p.write_text(s)
