import pathlib
# RE-ANCHORED 2026-09-06: the code it patches moved out of app/ticker.go when
# that 978-line file was split along the role boundaries the architecture
# already defined (MVS-D-77/S-7). A pure move — the mutation is unchanged.
# A severity tie decides the burst's tone by POSITION again, which is the
# mechanism MVS-D-73 replaced. Severity is a three-value colour tier, so two
# hazards of different kinds share one constantly — a red hurricane and a red
# tornado warning in one burst would sound the low sweep or the EAS dual-tone
# depending on nothing a listener could reason about.
#
# RE-ANCHORED AT T3.10b. The branch was deleted outright before; with the
# takeover's read gone from this file, worstOf became the only user of `cast`
# here and removing the call left the tree UNCOMPILABLE — a verdict that is no
# evidence either way. Disabling the branch is the same defect and still builds.
p = pathlib.Path("app/burst_words.go"); s = p.read_text()
old = "\t\tif cast.Classify(e.Type).ToneRank() > cast.Classify(worst.Type).ToneRank() {"
new = "\t\tif false && cast.Classify(e.Type).ToneRank() > cast.Classify(worst.Type).ToneRank() {"
assert old in s, "mG4"
p.write_text(s.replace(old, new, 1))
