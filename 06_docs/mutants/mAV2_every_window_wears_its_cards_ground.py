import pathlib
# The narrow reading is dropped: EVERY card hands its ground to its window, so a
# location report's window floats on CardBG instead of the modal tone. Nothing is
# unreadable and nothing looks broken — which is the point. It restyles every
# report window in the app to say something the ground does not mean, and only a
# test that states the narrow rule can tell the two apart.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """	if c.Slot != lineup.BreakingAlert {
		return ""
	}
"""
assert old in s, "mAV2"
p.write_text(s.replace(old, "", 1))
