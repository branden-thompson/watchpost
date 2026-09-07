import pathlib
# The two within-rung dimensions swapped. The newest alert leads its rung
# instead of the worst one.
p = pathlib.Path("platform/lineup/plan.go"); s = p.read_text()
old = """		if p[i].arrival.Severity != p[j].arrival.Severity {
			return p[i].arrival.Severity > p[j].arrival.Severity
		}
		if !p[i].arrival.At.Equal(p[j].arrival.At) {
			return p[i].arrival.At.After(p[j].arrival.At)
		}"""
new = """		if !p[i].arrival.At.Equal(p[j].arrival.At) {
			return p[i].arrival.At.After(p[j].arrival.At)
		}
		if p[i].arrival.Severity != p[j].arrival.Severity {
			return p[i].arrival.Severity > p[j].arrival.Severity
		}"""
assert old in s, "m76"
p.write_text(s.replace(old, new, 1))
