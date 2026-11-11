import pathlib
# 0.19.5 D-38: the follow-ups only ever mentioned never reach the index, so
# the code's citations of F-97 and the rest resolve nowhere.
# Killed by TestEveryFollowUpTheCodeCitesIsIndexed.
p = pathlib.Path('tools/ids/main.go'); s = p.read_text()
old = '\trows = append(rows, followUpRows(rows, all)...)\n'
assert s.count(old) == 1, 'm811'
p.write_text(s.replace(old, '\t_ = followUpRows\n\t_ = all\n', 1))
