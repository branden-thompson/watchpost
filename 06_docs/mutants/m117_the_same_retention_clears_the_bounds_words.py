import pathlib
# D-143 (0.19.0 W1.2): only a changed retention clears the bound's words; Settings applies the same one on every close.
# Killed by TestAChangedRetentionClearsTheBoundsWords.
p = pathlib.Path("platform/history/history.go"); s = p.read_text()
old = "\t\tif d.Hours != hours || d.Days != days {\n\t\t\tdelete(s.bounded, name)"
assert old in s, "m117"
p.write_text(s.replace(old, "\t\tif true {\n\t\t\tdelete(s.bounded, name)"))
