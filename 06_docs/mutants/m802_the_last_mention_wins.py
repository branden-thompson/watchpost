import pathlib
# 0.19.5 D-35: a later mention replaces a better one, so an MVS-D ID points at
# wherever it was last cited instead of where it was ruled.
# Killed by TestAnMVSRulingIsIndexedByItsBestMention.
p = pathlib.Path('tools/ids/main.go'); s = p.read_text()
old = '\tif held, ok := into[num]; ok && held.rank <= m.rank {\n'
assert s.count(old) == 1, 'm802'
p.write_text(s.replace(old, '\tif held, ok := into[num]; ok && held.rank < 0 {\n', 1))
