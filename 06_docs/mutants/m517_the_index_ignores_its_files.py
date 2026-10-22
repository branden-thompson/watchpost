import pathlib
# 0.19.0 D-242: recording's index answers though the day's files changed, missing another instance's issue.
# Killed by TestRecordingHoldsNoDocument.
p = pathlib.Path('platform/history/history.go'); s = p.read_text()
old = '\tif held, _, ok := s.index.Last(f.day); ok && held.files == now {\n'
assert s.count(old) == 1, 'm517'
p.write_text(s.replace(old, '\tif held, _, ok := s.index.Last(f.day); ok {\n', 1))
