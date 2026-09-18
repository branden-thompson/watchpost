import pathlib
# The bed goes back to tuning through `tuneCallsign`, which searches the mount
# list the LISTENER's last tune left behind and returns in SILENCE when the
# callsign is not in it. The operator presses the key, the row says the relay is
# tuned, and the station carries dead air (D-117).
p = pathlib.Path("app/bedrelay.go"); s = p.read_text()
old = """			lp.deck.tuneResolved(chosen, relays)"""
assert old in s, "mAK1"
p.write_text(s.replace(old, """			lp.deck.tuneCallsign(chosen.Callsign)""", 1))
