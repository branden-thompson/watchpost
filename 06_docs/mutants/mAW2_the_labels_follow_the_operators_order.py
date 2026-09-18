import pathlib
# The labels come back in the order the bits happen to sit rather than the
# registry's — which today is the same thing, so this is a mutation that has to
# be written to actually reverse it. Two operators choosing the same three
# reports would then produce two different strings, and a set that cannot be
# compared cannot be memoised, goldened, or read back as the same card.
p = pathlib.Path("platform/report/report.go"); s = p.read_text()
old = """	for _, k := range s.Kinds() { // bounded by the registry (P10-02)
		out = append(out, Of(k).Label)
	}"""
new = """	ks := s.Kinds()
	for i := len(ks) - 1; i >= 0; i-- { // bounded by the registry (P10-02)
		out = append(out, Of(ks[i]).Label)
	}"""
assert old in s, "mAW2"
p.write_text(s.replace(old, new, 1))
