import pathlib
# 0.19.3 D-366: a reset clears the ink.
# Killed by TestInkAtReadsTheCellsForeground.
p = pathlib.Path('platform/render/ink.go'); s = p.read_text()
old = '\t\tcase n == 0:\n\t\t\tpen = Cell{}\n'
assert s.count(old) == 1, 'm646'
p.write_text(s.replace(old, '\t\tcase n == 0:\n\t\t\tpen.Paper, pen.Papered = color.RGBA{}, false\n', 1))
