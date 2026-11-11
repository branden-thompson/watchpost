import pathlib
# 0.19.5 Q2.2 (RN-2): enabling the last muted class turns mute mode off.
# Killed by TestEnablingTheLastMutedClassTurnsMuteModeOff.
p = pathlib.Path('modes/tty/setup_tones.go'); s = p.read_text()
old = '\tif len(next) == 0 {\n\t\td.setup.toneMode = toneModeOn\n'
assert s.count(old) == 1, 'm684'
p.write_text(s.replace(old, '\tif len(next) < 0 {\n\t\td.setup.toneMode = toneModeOn\n', 1))
