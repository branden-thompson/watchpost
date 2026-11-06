import pathlib
# 0.19.5 D-16 (DEF-4): a sweep's record is named from the newest release tag.
# Killed by TestTheReleasePathsAreDerivedFromTheLastTag.
p = pathlib.Path('Makefile'); s = p.read_text()
old = 'VERDICTS_RECORD := 06_docs/mutants/verdicts/since-v$(LAST_TAG).log\n'
assert s.count(old) == 1, 'm673'
p.write_text(s.replace(old, 'VERDICTS_RECORD := 06_docs/02_features/0.16.0-broadcaster-ui/07-readiness/mutant-verdicts.log\n', 1))
