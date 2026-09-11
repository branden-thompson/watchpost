import pathlib
# The pre-build removed from the settle: the next card is only prepared once the
# air is free, so every cutover pays the 1.03 s build in silence.
#
# RE-ANCHORED AT D-82: the air is asked about a LANE now, and "is anything
# reading anywhere" is `anyOnAir`. The mutation is unchanged — it is the
# condition this rule is NOT allowed to have.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """\td, air := d.takeTheAir()
\td, prep := d.prepareNext()"""
new = """\td, air := d.takeTheAir()
\tvar prep []Effect
\tif !d.lineup.anyOnAir() {
\t\td, prep = d.prepareNext()
\t}"""
assert old in s, "m91"
p.write_text(s.replace(old, new, 1))
