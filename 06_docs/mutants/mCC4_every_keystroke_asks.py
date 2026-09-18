import pathlib
# D-130. The pause is gone: every keystroke asks. For a query the embedded index
# does not hold this is a GEOCODER CALL PER KEY — which is the rule setup.go
# already carries (AI-8, ToS: never the network per keystroke) and the reason the
# HUM LEAD ruled the debounce in the first place.
p = pathlib.Path("platform/debounce/debounce.go"); s = p.read_text()
old = "const Pause = 300 * time.Millisecond"
# THE UNIT IS KEPT so `time` stays used; a bare 0 fails to build, which is
# INVALID rather than a surviving rule.
new = "const Pause = 0 * time.Millisecond"
assert old in s, "mCC4"
p.write_text(s.replace(old, new, 1))
