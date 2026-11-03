import pathlib
# 0.19.3 AX-5 (A-107 (7)): without the picture the status is said.
# Killed by TestWithoutThePictureTheStatusAndTimelineAreText.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = "\tfor _, said := range []string{d.edgeWords(), d.mapStatusText()} {\n"
assert s.count(old) == 1, 'm634'
p.write_text(s.replace(old, "\tfor _, said := range []string{d.edgeWords()} {\n", 1))
