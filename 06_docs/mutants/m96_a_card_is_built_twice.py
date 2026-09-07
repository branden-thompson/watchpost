import pathlib
# The standby transition dropped from prepareNext, so the card stays admitted
# and every tick describes its build again.
#
# RE-ANCHORED three times: BD-8 added the Slot, F-D1 moved the Set, and F-D1
# round 2 put a progress check beside it.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """\t\tmoved, err := d.lineup.Set(standby)
\t\tif err != nil {
\t\t\treturn d, nil
\t\t}"""
new = """\t\tmoved := d.lineup"""
assert old in s, "m96"
p.write_text(s.replace(old, new, 1))
