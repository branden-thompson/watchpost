import pathlib
# D-143 (0.19.0 W1.2): every version counts toward the bound, so every version is trimmed, an old one first.
# Killed by TestAnOldVersionCountsAndGoesFirst.
p = pathlib.Path("platform/history/history.go"); s = p.read_text()
old = "\tfor _, v := range readDirs(filepath.Join(s.root, d.Name), visits) { // its versions (P10-02)"
assert old in s, "m118"
p.write_text(s.replace(old, "\tfor _, v := range []string{\"v\" + strconv.Itoa(d.Version)} { // its versions (P10-02)"))
