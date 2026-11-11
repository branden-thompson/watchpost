import pathlib
# 0.19.5 D-33: in Settings, p previews a voice only on a voice (cast) row.
# Killed by TestOnlyAVoiceRowPreviews.
p = pathlib.Path('modes/tty/setup.go'); s = p.read_text()
old = '\t\tif picker && setupTable()[d.setup.focus].group == groupCast {'
assert s.count(old) == 1, 'm799'
p.write_text(s.replace(old, '\t\tif picker {', 1))
