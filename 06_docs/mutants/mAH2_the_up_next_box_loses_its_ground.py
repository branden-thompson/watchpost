import pathlib
# The UP NEXT box goes back to no ground of its own, so the one card the operator
# READS sits on whatever the frame happens to be painted on (D-114, HUM LEAD
# 2026-09-13: "the UP Next Box probably needs a bkg color other than none").
#
# THE TINT IS DROPPED BY MAKING IT A NO-OP rather than by deleting the loop:
# removing the loop leaves `ground` unused and the mutation does not compile,
# which is INVALID — no evidence either way — rather than a measurement.
p = pathlib.Path("modes/tty/broadcaster_upnext.go"); s = p.read_text()
old = """	ground := fg + ";" + bg"""
assert old in s, "mAH2"
p.write_text(s.replace(old, """	ground := ""
	_, _ = fg, bg""", 1))
