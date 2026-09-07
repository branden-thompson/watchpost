# RE-ANCHORED twice: at T3.8's completion (a card's words became a SCRIPT in
# parts) and at the T3.10 red team (BurstHead and DivertNotice were retired, so
# Transition is the one structural slot left). The rule is unchanged.
import pathlib
# The rule moved back to Propose alone: Queue and Set stop enforcing it, so a
# wordless structural card queued directly reaches standby, describes no build,
# and stands there for ever with the rail stopped behind it.
p = pathlib.Path("platform/lineup/card.go"); s = p.read_text()
old = """\tif err := invariant.Check(c.Words() != "" || c.Slot != Transition,
\t\t"a card whose words are fixed at proposal never exists without them"); err != nil {
\t\treturn err
\t}
"""
assert old in s, "mD9"
p.write_text(s.replace(old, "", 1))
