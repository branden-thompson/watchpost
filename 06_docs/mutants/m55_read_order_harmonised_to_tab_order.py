import pathlib
# The tidy-up a future reader performs: "why are there three orderings? make
# read order just follow the tabs." It silently changes what is spoken.
p = pathlib.Path("platform/category/category.go"); s = p.read_text()
old = """	byRank := map[int]Category{}
	for _, c := range All() {
		if s := Of(c); s.ReadRank > 0 {
			byRank[s.ReadRank] = c
		}
	}"""
new = """	byRank := map[int]Category{}
	for i, c := range All() {
		byRank[i+1] = c
	}"""
assert old in s, "m55"
p.write_text(s.replace(old, new, 1))
