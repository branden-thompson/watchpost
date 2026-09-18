import pathlib
# A standing-by card whose data has been superseded is never asked for again, so
# it is dropped as it takes the air (PD-3) and the station opens by apologising:
# "That report is out of date and has been dropped."
#
# The operator sets up a line-up, leaves it fifteen minutes, presses the key, and
# the first thing the listener hears is the apology. It is the defect D-84 itself
# creates by letting the Composer work on standby.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "\t\t\treturn d.refreshStandby()\n"
new = "\t\t\treturn d, nil\n"
assert old in s, "mP2"
p.write_text(s.replace(old, new, 1))
