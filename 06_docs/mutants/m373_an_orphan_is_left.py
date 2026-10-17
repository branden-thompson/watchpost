import pathlib
# 0.19.0 FR-9.2: an orphan proven watchpost's is ended at start.
# Killed by TestAnOrphanedSayIsEndedAtStart.
p = pathlib.Path('domains/radio/synth/children.go'); s = p.read_text()
old = '\t\tcase orphanOf(p, uid, started) && procs.End(n) == nil:\n'
assert old in s, 'm373'
p.write_text(s.replace(old, '\t\tcase false && orphanOf(p, uid, started) && procs.End(n) == nil:\n'))
