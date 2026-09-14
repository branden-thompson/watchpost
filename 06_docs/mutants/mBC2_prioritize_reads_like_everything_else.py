import pathlib
# PRIORITIZE loses its weight and its tone and reads as ordinary text. It is the
# ONE choice in this window that moves every other card, and nothing on the row
# would say so — the operator's eye has no reason to stop there before pressing
# space (HUM LEAD, 2026-09-14).
p = pathlib.Path("modes/tty/request.go"); s = p.read_text()
old = """		render.Bold(render.Tint("PRIORITIZE", render.Tok(render.NameAdvisory)))+"""
assert old in s, "mBC2"
p.write_text(s.replace(old, """		"PRIORITIZE"+""", 1))
