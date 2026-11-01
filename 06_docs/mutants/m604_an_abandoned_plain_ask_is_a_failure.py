import pathlib
# 0.19.1 D-354: the propagation client's ask its caller cancelled is not the host failing.
# Killed by TestAnAbandonedPlainAskIsNotAFailure.
p = pathlib.Path('platform/httpx/plain.go'); s = p.read_text()
old = '\tresp, err := p.http.Do(req)\n\tif err != nil {\n\t\tif !Abandoned(ctx) {\n'
assert s.count(old) == 1, 'm604'
p.write_text(s.replace(old, '\tresp, err := p.http.Do(req)\n\tif err != nil {\n\t\tif !Abandoned(ctx) || true {\n', 1))
