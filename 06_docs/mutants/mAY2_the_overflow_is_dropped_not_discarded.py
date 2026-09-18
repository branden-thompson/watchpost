import pathlib
# The card pushed off the bottom is simply deleted instead of going onto the
# discard pile. The running order looks right and the card is GONE — the Producer
# cannot re-request a copy and the operator has no record it ever existed, which
# is the one thing the ruling asked for.
p = pathlib.Path("platform/lineup/operator.go"); s = p.read_text()
old = "		out = out.discard(fallen)"
assert old in s, "mAY2"
p.write_text(s.replace(old, "		_ = fallen", 1))
