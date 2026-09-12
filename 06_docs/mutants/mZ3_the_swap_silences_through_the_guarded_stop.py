import pathlib
# The surface swap silences the monitor through the GUARDED `Stop` instead of
# `stopMonitor`. It runs after `owner` has already moved to the console, so the
# guard refuses it and Observer's audio keeps playing under the Broadcaster —
# which is the HUM LEAD's original complaint ("audio in Broadcaster is still
# pulling audio from Observer") arriving by a new road (D-91).
p = pathlib.Path("app/airscope.go"); s = p.read_text()
old = "\tlp.deck.stopMonitor()"
new = "\tlp.deck.Stop()"
assert old in s, "mZ3"
p.write_text(s.replace(old, new, 1))
