import pathlib
# 0.19.5 Q2.1 (RN-1): the attention tone plays aside, never through the bars.
# Killed by TestAMutedClassSoundsNoToneAndAToneNeverDrivesTheBars.
p = pathlib.Path('app/radio.go'); s = p.read_text()
old = '\t_ = d.engine.PreviewAside(synth.ToneRate, bytes.NewReader(tone))'
assert s.count(old) == 1, 'm676'
p.write_text(s.replace(old, '\t_ = d.engine.Preview(synth.ToneRate, bytes.NewReader(tone))', 1))
