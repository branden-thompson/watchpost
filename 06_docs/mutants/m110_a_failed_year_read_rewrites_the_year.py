import pathlib
# #27 (0.19.0 FR-6.5, W1.1): a year's roll-up that exists but cannot be read is never written over.
# Killed by TestAFailedYearReadNeverRewritesTheYear.
p = pathlib.Path("platform/history/history.go"); s = p.read_text()
old = "	if _, err := os.Stat(yp); !ok && err == nil {"
assert old in s, "m110"
p.write_text(s.replace(old, "	if _, err := os.Stat(yp); !ok && err == nil && false {"))
