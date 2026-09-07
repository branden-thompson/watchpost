import pathlib
# The duplicate lookup asks the wrong question — find("") never matches, so no
# duplicate is ever detected. Set and Remove become ambiguous, and an ambiguous
# Remove is a card read twice.
p = pathlib.Path("platform/lineup/lineup.go"); s = p.read_text()
old = "\t_, _, taken := l.find(c.ID)"
new = '\t_, _, taken := l.find("")'
assert old in s, "m68"
p.write_text(s.replace(old, new, 1))
