import pathlib
# 0.19.0 R2H-3: the release workflow drops its finaltag step, so a final tag ships on a release candidate.
# Killed by TestTheReleaseRunsFinaltag.
p = pathlib.Path('.github/workflows/release.yml'); s = p.read_text()
old = '      - name: a final tag ships on final libraries (D-3, RH-9)\n        run: go run ./tools/finaltag "$GITHUB_REF_NAME"\n'
assert s.count(old) == 1, 'm555'
p.write_text(s.replace(old, '', 1))
