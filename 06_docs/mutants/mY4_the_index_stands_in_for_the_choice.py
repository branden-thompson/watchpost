import pathlib
# The operator's selection is read off `bedPick` — the INDEX — instead of the relay
# line that was actually tuned. `bedPick`'s zero value is a real relay, so a station
# nobody has touched reports its nearest transmitter as though the operator had
# selected it, and the row claims a tune that never happened.
p = pathlib.Path("app/bedrelay.go"); s = p.read_text()
old = """	lp.mu.Lock()
	defer lp.mu.Unlock()
	return lp.bedRelay
}"""
new = """	relays := lp.bedRelays()
	lp.mu.Lock()
	defer lp.mu.Unlock()
	if lp.bedPick >= len(relays) {
		return ""
	}
	return relayLine(relays[lp.bedPick])
}"""
assert old in s, "mY4"
p.write_text(s.replace(old, new, 1))
