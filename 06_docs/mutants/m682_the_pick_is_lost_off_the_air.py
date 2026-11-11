import pathlib
# 0.19.5 Q2.1 (RN-1): [m] saves the pick whatever holds the air.
# Killed by TestTheSourcePickIsAlwaysSaved.
p = pathlib.Path('app/radio.go'); s = p.read_text()
old = '\tif persist != nil {\n\t\t_ = persist(mode)'
assert s.count(old) == 1, 'm682'
p.write_text(s.replace(old, '\tif persist != nil && air {\n\t\t_ = persist(mode)', 1))
