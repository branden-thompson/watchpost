import pathlib
# 0.19.0 F-219: a hung say's report is read past its main thread, every thread logged as one.
# Killed by TestAHangsReportNamesItsMainThread.
p = pathlib.Path('domains/radio/synth/synth_test.go'); s = p.read_text()
old = '\t\tif i > 0 && !strings.HasPrefix(l, "    +") {\n'
assert s.count(old) == 1, 'm542'
p.write_text(s.replace(old, '\t\tif i > 0 && false {\n', 1))
