import pathlib
# The band record grows with the day (P10-03).
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = '\tif len(b.lines) > bandRecordCap {\n\t\tb.lines = b.lines[len(b.lines)-bandRecordCap:]\n\t}\n'
new = '\tif len(b.lines) > bandRecordCap*bandRecordCap {\n\t\tb.lines = b.lines[len(b.lines)-bandRecordCap:]\n\t}\n'
assert old in s, "mC5"
p.write_text(s.replace(old, new, 1))
