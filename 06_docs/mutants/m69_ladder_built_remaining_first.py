import pathlib
# The two bands composed in the other order. Thirteen rungs still, so the
# ladder's own count invariant is satisfied — and every alert is read in the
# wrong band's order.
p = pathlib.Path("platform/lineup/plan.go"); s = p.read_text()
old = """	for _, c := range read { // bounded by the registry (P10-02)
		out = append(out, Rung{Band: CloseBand, Category: c})
	}
	for _, c := range read {
		if c == category.Emergency {
			continue
		}
		out = append(out, Rung{Band: RemainingBand, Category: c})
	}"""
new = """	for _, c := range read {
		if c == category.Emergency {
			continue
		}
		out = append(out, Rung{Band: RemainingBand, Category: c})
	}
	for _, c := range read { // bounded by the registry (P10-02)
		out = append(out, Rung{Band: CloseBand, Category: c})
	}"""
assert old in s, "m69"
p.write_text(s.replace(old, new, 1))
