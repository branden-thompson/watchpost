import pathlib
# 0.19.5 Q2.4 (RN-5): a seismic state is unknown until the feed answers.
# Killed by TestASeismicReportIsUnknownUntilTheFeedAnswers.
p = pathlib.Path('app/seismic.go'); s = p.read_text()
old = '\tif ss == nil || ss.AsOf.IsZero() {\n'
assert s.count(old) == 1, 'm689'
p.write_text(s.replace(old, '\tif ss == nil {\n', 1))
