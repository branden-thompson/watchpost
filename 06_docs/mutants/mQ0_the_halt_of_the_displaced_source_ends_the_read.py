import pathlib
# The read stops asking whether its OWN audio has begun, so the Stopped that
# `halt` emits — for the source `StartSource` displaced, one millisecond before
# this read's first word — comes home as this read ending.
#
# THE DEFECT THE HUM LEAD REPORTED TWICE. Every card is then Failed{Routed},
# discarded, and its location benched for five minutes (D-67); at twenty-five
# pool entries the console reads "waiting for the line-up" in every slot, and the
# station is silent. MEASURED: the read returned false in ONE MILLISECOND, with
# the engine reporting a Stopped whose Name was empty.
p = pathlib.Path("app/mainread.go"); s = p.read_text()
old = """	case !r.begun():
		// A TERMINAL STATUS FOR SOMETHING ELSE (F-95). It belongs to the source
		// this read displaced — `StartSource` halts it first, and `halt` ends
		// with a Stopped. Ending here is ending a read that has not begun.
"""
assert old in s, "mQ0"
p.write_text(s.replace(old, "", 1))
