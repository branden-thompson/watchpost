import pathlib
# A save that chose nothing writes the BORROWED epicentre back as the station's
# own, silently ending the D-72 fallback: the station stops following the
# listener's default location the first time anyone opens Settings and presses
# enter, and nothing on screen says so (D-115).
p = pathlib.Path("modes/tty/setup.go"); s = p.read_text()
old = """	set := d.cfg.SetTransmitter
	if set == nil || d.setup.txRef == nil {
		return nil
	}
	next := *d.setup.txRef"""
assert old in s, "mAI3"
new = """	set := d.cfg.SetTransmitter
	if set == nil {
		return nil
	}
	next := snapshot.LocationRef{}
	if d.setup.txRef != nil {
		next = *d.setup.txRef
	} else if cur := d.currentTransmitter(); cur != nil {
		next = *cur
	}"""
p.write_text(s.replace(old, new, 1))
