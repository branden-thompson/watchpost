import pathlib
# The relay-fault window's cursor and clock leave the modal memo key, so the
# frame is rendered ONCE and replayed for as long as the window is open.
#
# Everything underneath stays correct — the arrows move the cursor, the clock
# runs down, the fall-through fires on time — and the listener sees a still
# picture. It took three UAT rounds to find, because every test called
# renderModal (the memo's MISS path) rather than modalView.
p = pathlib.Path("modes/tty/memo.go"); s = p.read_text()
old = "\t\tk.faultFocus, k.faultLeft = d.relayFault.focus, d.relayFault.left\n"
assert old in s, "mV3"
p.write_text(s.replace(old, "", 1))
