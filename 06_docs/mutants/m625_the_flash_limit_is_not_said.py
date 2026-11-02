import pathlib
# 0.19.3 D-361: the speeds past the flash ceiling say so.
# Killed by TestTheFastSpeedsSayTheFlashLimit.
p = pathlib.Path('modes/tty/map_playback.go'); s = p.read_text()
old = "\tif c < flashCeiling {\n"
assert s.count(old) == 1, 'm625'
p.write_text(s.replace(old, "\tif c < 0 {\n", 1))
