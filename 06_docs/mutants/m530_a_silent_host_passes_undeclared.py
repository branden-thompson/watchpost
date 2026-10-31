import pathlib
# 0.19.0 D-254 (RC-2): a host whose say renders nothing is NOT RUN without being declared mute.
# Killed by TestASilentHostFailsUnlessDeclaredMute.
p = pathlib.Path('domains/radio/synth/synth_test.go'); s = p.read_text()
old = '\tif declared == "" {\n\t\treturn false, why'
assert s.count(old) == 1, 'm530'
p.write_text(s.replace(old, '\tif false {\n\t\treturn false, why', 1))
