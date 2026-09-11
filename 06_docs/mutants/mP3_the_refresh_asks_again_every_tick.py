import pathlib
# The ask no longer moves the stamp, so the same card is sent to the Composer on
# EVERY tick until its words come home — one build a second against the provider,
# for as long as the build takes. The schedule ticks at 1 s and a cold build is
# 1.03 s, so this is a storm by construction rather than by bad luck.
p = pathlib.Path("platform/lineup/stale.go"); s = p.read_text()
old = "\t\tbumped := c\n\t\tbumped.BuiltAt = d.now\n\t\tnext, err := d.lineup.Set(bumped)\n\t\tif err != nil {\n\t\t\treturn d, nil\n\t\t}\n\t\td.lineup = next\n"
assert old in s, "mP3"
p.write_text(s.replace(old, "", 1))
