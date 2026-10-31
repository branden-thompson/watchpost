import pathlib
# 0.19.0 I-3: an orphan whose end fails keeps its record, so the next start tries again.
# Killed by TestAnOrphanThatCannotBeEndedIsTriedAgain.
p = pathlib.Path('domains/radio/synth/children.go'); s = p.read_text()
old = '\t\tcase orphanOf(p, uid, started):\n\t\t\tfailed++ // kept: the next start tries again\n'
assert old in s, 'm426'
p.write_text(s.replace(old, '\t\tcase orphanOf(p, uid, started):\n\t\t\tfailed++\n\t\t\t_ = os.Remove(path)\n'))
