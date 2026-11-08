import pathlib
# 0.19.5 D-35: the index leaves the MVS-D section out, so the code's 85
# multi-voice citations resolve nowhere.
# Killed by TestEveryMVSRulingTheCodeCitesIsIndexed.
p = pathlib.Path('tools/ids/main.go'); s = p.read_text()
old = '\treturn render(rows, releases(docs)) + renderMVS(mvs, releases(docs)), nil\n'
assert s.count(old) == 1, 'm803'
p.write_text(s.replace(old, '\treturn render(rows, releases(docs)) + renderMVS(nil, releases(docs)), nil\n', 1))
