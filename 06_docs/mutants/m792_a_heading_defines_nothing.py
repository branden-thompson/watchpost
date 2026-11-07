import pathlib
# 0.19.5 Q7.6 (D-11): a heading that opens with an ID defines it, the form 0.16.0 rules in.
# Killed by TestARowDefinesAnIDByItsFirstCell.
p = pathlib.Path('tools/ids/main.go'); s = p.read_text()
old = '\t\tif h := definingHeading.FindStringSubmatch(line); h != nil {\n'
assert s.count(old) == 1, 'm792'
p.write_text(s.replace(old, '\t\tif h := definingHeading.FindStringSubmatch(line); h != nil && false {\n', 1))
