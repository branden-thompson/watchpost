# RE-ANCHORED AT T3.10b: a takeover's words are a SCRIPT composed from the card's
# refs, not one line built from one event, so "nothing to say" is Script.Empty().
import pathlib
# A script that renders nothing is handed to the schedule as a Built with no words,
# which the Director's own invariant refuses — the card sits at standby for ever.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = '\t\tif sc.Empty() {\n\t\t\treturn x.decline(v, v.ID, "the script rendered nothing to say")\n\t\t}\n'
new = '\t\tif sc.Empty() && !sc.Empty() {\n\t\t\treturn x.decline(v, v.ID, "the script rendered nothing to say")\n\t\t}\n'
assert old in s, "mC1"
p.write_text(s.replace(old, new, 1))
