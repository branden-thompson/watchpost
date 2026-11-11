import pathlib
# 0.19.5 D-37: a mention another release qualifies ("MVS-D-9", "0.18.0 D-8")
# is indexed as the untabled release's own.
# Killed by TestAnUntabledRulingIsIndexedByItsMention.
p = pathlib.Path('tools/ids/main.go'); s = p.read_text()
old = '\t\t\t\tif defined[num] || qualified(line[:m[0]]) {\n'
assert s.count(old) == 1, 'm807'
p.write_text(s.replace(old, '\t\t\t\tif defined[num] {\n', 1))
