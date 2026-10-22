import pathlib
# 0.19.0 D-242: the days held for readers are bounded by count alone.
# Killed by TestRecordingHoldsNoDocument.
p = pathlib.Path('platform/history/history.go'); s = p.read_text()
old = 'MaxBytes: 8 << 20}'
assert s.count(old) == 1, 'm515'
p.write_text(s.replace(old, 'MaxBytes: 0}', 1))
