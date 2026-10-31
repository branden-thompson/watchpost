import pathlib
# 0.19.0 FR-5.2: an update ended by leaving the mode is no failure.
# Killed by TestTheStatusIsSaidWithoutThePicture.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tif err == nil || errors.Is(err, context.Canceled) {\n'
assert old in s, 'm338'
p.write_text(s.replace(old, '\tif err == nil || errors.Is(err, context.DeadlineExceeded) {\n'))
