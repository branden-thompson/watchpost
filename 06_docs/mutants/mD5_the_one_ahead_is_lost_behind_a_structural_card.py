# RE-ANCHORED AT T3.8's completion: a card's words are a SCRIPT in parts, not a
# string (MVS-D-77). Card.Text became Card.Words, and Speak/Built carry a Script.
# The rules are unchanged; only the words they are written against moved.
import pathlib
# The walk stops at the first card that needs no build instead of looking behind
# it, so a burst opening with a head AND a transition leaves its first alert
# unprepared and that alert\'s 1.03 s build lands as dead air (DR-7).
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """\t\t\tcontinue
\t\t}
\t\tif err := invariant.Check(standby.Script.Empty(), "a card waiting to be built has no words yet"); err != nil {"""
new = """\t\t\treturn d, nil
\t\t}
\t\tif err := invariant.Check(standby.Script.Empty(), "a card waiting to be built has no words yet"); err != nil {"""
assert old in s, "mD5"
p.write_text(s.replace(old, new, 1))
