import pathlib
# The release stops asking which lane it belongs to, so a REPORT leaving the air
# gives back a band it never took — clearing the callout for a tornado warning
# still being read over it.
#
# This is F-71, which `clearBand` carried in a comment for two releases with its
# trigger named exactly: benign only while ONE card held the air at a time. D-82
# put a hazard and a report on the air together, which is that trigger.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = """\tif v.Track != lineup.AlertRail {
\t\treturn nil // it never held the band; giving back what it did not take would take it from the rail
\t}
"""
assert old in s, "mN3"
p.write_text(s.replace(old, "", 1))
