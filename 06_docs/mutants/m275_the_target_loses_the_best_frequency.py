import pathlib
# 0.19.0 D-197: the Broadcaster's target names its best frequency.
# Killed by TestTheTargetSaysWhatIsUsable.
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = 'if best := propBestWords(limitsOf(bands), true); best != "" {'
assert old in s, 'm275'
p.write_text(s.replace(old, 'if best := propBestWords(limitsOf(bands), true); false && best != "" {'))
