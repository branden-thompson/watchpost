import pathlib
# D-134. The station's live state loses its weight, so *** ON AIR · BROADCASTING
# *** reads at the same emphasis as STANDBY and STOPPED — and the one state the
# operator must never mistake is the one that stops standing out (HUM LEAD,
# 2026-09-15).
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = '		state = render.Bold(render.Tint("*** ON AIR "+g.Dot+" BROADCASTING ***", render.Tok(render.AlertModalText)))'
new = '		state = render.Tint("*** ON AIR "+g.Dot+" BROADCASTING ***", render.Tok(render.AlertModalText))'
assert old in s, "mCG1"
p.write_text(s.replace(old, new, 1))
