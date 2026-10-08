import pathlib
# FR-6.5 (0.19.0 W1.2): values held compactly are written as encoding/json wrote 0.18.0's, exponents included.
# Killed by TestValuesAreWrittenAsBefore.
p = pathlib.Path("platform/history/history.go"); s = p.read_text()
old = "	if abs := math.Abs(x); abs != 0 && (abs < 1e-6 || abs >= 1e21) {"
assert old in s, "m114"
p.write_text(s.replace(old, "	if abs := math.Abs(x); abs != 0 && abs < 1e-6 {"))
