import pathlib
# The other side of the ratified boundary: the window loosened, so a disaster
# past 24:01 still leads the burst. m74 moves it the other way; between them
# both instants the HUM LEAD named are pinned.
p = pathlib.Path("platform/lineup/plan.go"); s = p.read_text()
old = "	if now.Sub(at) <= DisasterFreshWindow {"
new = "	if now.Sub(at) <= DisasterFreshWindow+time.Hour {"
assert old in s, "m79"
p.write_text(s.replace(old, new, 1))
