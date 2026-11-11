import pathlib
# 0.19.5 Q4.1 (PF-10): Apply replaces a section's hours and never writes into
# them, since published snapshots share them.
# Killed by TestAPublishedForecastIsNeverWrittenAfterwards.
p = pathlib.Path('platform/snapshot/assembler.go'); s = p.read_text()
old = '\tif pd.Hourly != nil {\n\t\tsec.Hourly = append([]Hourly(nil), pd.Hourly...)\n\t}\n'
assert s.count(old) == 1, 'm748'
p.write_text(s.replace(old, '\tif pd.Hourly != nil && len(sec.Hourly) == len(pd.Hourly) {\n\t\tcopy(sec.Hourly, pd.Hourly)\n\t} else if pd.Hourly != nil {\n\t\tsec.Hourly = append([]Hourly(nil), pd.Hourly...)\n\t}\n', 1))
