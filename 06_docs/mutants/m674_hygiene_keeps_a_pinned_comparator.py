import pathlib
# 0.19.5 D-16: hygiene keeps the newest release tag's binary as the comparator.
# Killed by TestTheReleasePathsAreDerivedFromTheLastTag.
p = pathlib.Path('Makefile'); s = p.read_text()
old = 'HYGIENE_KEEP := watchpost watchpost-$(LAST_TAG) mutant-verdicts.timings\n'
assert s.count(old) == 1, 'm674'
p.write_text(s.replace(old, 'HYGIENE_KEEP := watchpost watchpost-0.14.1 mutant-verdicts.timings\n', 1))
