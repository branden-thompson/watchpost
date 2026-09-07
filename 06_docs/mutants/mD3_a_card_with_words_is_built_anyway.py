# RE-ANCHORED AT T3.8's completion: a card's words are a SCRIPT in parts, not a
# string (MVS-D-77). Card.Text became Card.Words, and Speak/Built carry a Script.
# The rules are unchanged; only the words they are written against moved.
import pathlib
# The structural branch deleted: a burst head whose words were fixed at proposal
# is handed a build that has nothing to compose, and WithText then refuses to
# put words on it — so it stands by for ever with the rail stopped behind it.
#
# RE-ANCHORED at F-D1 round 2: the branch asks the registry now.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """\t\tif !standby.Slot.textAtStandby() {
\t\t\t// Propose refuses such a card without its words, so there is
\t\t\t// nothing to build and nothing to wait for.
\t\t\tif err := invariant.Check(!standby.Script.Empty(), "a card whose words were fixed at proposal reaches standby carrying them"); err != nil {
\t\t\t\treturn d, nil
\t\t\t}
\t\t\tcontinue
\t\t}
"""
assert old in s, "mD3"
p.write_text(s.replace(old, "", 1))
