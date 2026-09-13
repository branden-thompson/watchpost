import pathlib
# The bed's chips go back to naming the bare arrows while the binding is shifted,
# so the row advertises a key that steps a different control — worse than no chip,
# because the operator would try it (D-111).
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """	return o.KeyCap("⇧←") + "  " + relay + "  " + o.KeyCap("⇧→")"""
assert old in s, "mAE2"
p.write_text(s.replace(old, """	return o.KeyCap("←") + "  " + relay + "  " + o.KeyCap("→")""", 1))
