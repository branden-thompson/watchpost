import pathlib
# 0.19.5 D-37: a first cell carrying a priority ("FR-3.8 (P1)") no longer
# defines its ID, so the requirement resolves only to a traceability row.
# Killed by TestAPriorityStillDefinesAndAReferenceIsNoSubject.
p = pathlib.Path('tools/ids/main.go'); s = p.read_text()
old = '[a-z]?(?:\\s*\\([^)]*\\))?$`)'
assert s.count(old) == 1, 'm809'
p.write_text(s.replace(old, '[a-z]?$`)', 1))
