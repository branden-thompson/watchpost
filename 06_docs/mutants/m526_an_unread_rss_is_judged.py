import pathlib
# 0.19.0 RC-1: an arm with an RSS that does not read is judged as 0 MB.
# Killed by TestAG1RunThatCannotBeJudgedFails.
p = pathlib.Path('tools/perfsum/g1.go'); s = p.read_text()
old = 'if !s.at.Before(from) && math.IsNaN(s.rssKB) {'
assert s.count(old) == 1, 'm526'
p.write_text(s.replace(old, 'if false {', 1))
