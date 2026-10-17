import pathlib
# 0.19.0 FR-9.1: Say holds its child while it runs.
# Killed by TestSayIsTrackedWhileItRuns.
p = pathlib.Path('domains/radio/synth/voice.go'); s = p.read_text()
old = '\tif err := startTracked(cmd); err != nil {'
assert old in s, 'm376'
p.write_text(s.replace(old, '\tif err := cmd.Start(); err != nil {'))
