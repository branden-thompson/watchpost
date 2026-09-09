import pathlib
# The relay-fault window's cursor and clock leave the modal memo key, so the
# frame is rendered ONCE and replayed for as long as the window is open.
#
# Everything underneath stays correct — the arrows move the cursor, the clock
# runs down, the fall-through fires on time — and the listener sees a still
# picture. It took three UAT rounds to find, because every test called
# renderModal (the memo's MISS path) rather than modalView.
#
# THE ANCHOR MOVED AT 0.15.0 (FR-6.4), when the clock's hold became a third
# thing the frame shows that moves. The mutant is unchanged in what it removes:
# all of the window's moving parts, in one line.
p = pathlib.Path("modes/tty/memo.go"); s = p.read_text()
old = "\t\tk.faultFocus, k.faultLeft, k.faultHeld = d.relayFault.focus, d.relayFault.left, d.relayFault.held\n"
assert old in s, "mV3"
p.write_text(s.replace(old, "", 1))
