import pathlib
# 0.19.5 Q3.2 (PF-2): a string that is not valid UTF-8 is measured whole, since
# an invalid byte is measured by what follows it.
# Killed by TestWidthMatchesStripThenMeasure.
p = pathlib.Path('platform/render/text.go'); s = p.read_text()
old = '\tif !utf8.ValidString(s) {\n'
assert s.count(old) == 1, 'm731'
p.write_text(s.replace(old, '\tif !utf8.ValidString(s) && false {\n', 1))
