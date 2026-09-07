# Play the stream before its volume is set: a tick of full-volume audio over an
# alert, even though the right scale is computed.
import pathlib
p = pathlib.Path("domains/radio/player/engine.go"); s = p.read_text()
old = """	p.SetVolume(e.volume * scale)
	e.mu.Unlock()
	if !hold { // a rendered report waits for the alert rather than opening under it
		p.Play()
	}"""
assert old in s, "m31"
p.write_text(s.replace(old, """	e.mu.Unlock()
	if !hold { // a rendered report waits for the alert rather than opening under it
		p.Play()
	}
	p.SetVolume(e.volume * scale)"""))
