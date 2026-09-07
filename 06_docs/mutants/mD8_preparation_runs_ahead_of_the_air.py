import pathlib
# The walk stops looking: a report standing by no longer ends it, so every
# completed build starts the next one and preparation runs arbitrarily far ahead
# of the air. A report composed several reads before it plays speaks data that
# was true when it was built (DR-7).
p = pathlib.Path("platform/lineup/lineup.go"); s = p.read_text()
old = """\t\t\tif c.State == Standby {
\t\t\t\treturn Card{}, t, false
\t\t\t}
"""
assert old in s, "mD8"
p.write_text(s.replace(old, "", 1))
