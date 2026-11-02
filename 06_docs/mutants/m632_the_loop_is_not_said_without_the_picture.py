import pathlib
# 0.19.3 AX-5 (A-107 (7)): without the picture the loop's line is text.
# Killed by TestWithoutThePictureTheStatusAndTimelineAreText.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = "\t\tif d.radarTimelineOn() {\n\t\t\tfor _, l := range render.WrapText(d.loopWords(), width-1) {"
assert s.count(old) == 1, 'm632'
p.write_text(s.replace(old, "\t\tif false {\n\t\t\tfor _, l := range render.WrapText(d.loopWords(), width-1) {", 1))
