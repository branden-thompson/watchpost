import pathlib
# A seam of the surface-to-app boundary loses its airBoundary row, so nothing says
# what it can do to the air. This is the failure mode the closed set exists to
# prevent: nineteen paths reached the engine and ONE of them asked, because no
# list said which owed a guard (D-91).
p = pathlib.Path("app/air_boundary_test.go"); s = p.read_text()
old = '\t"SetVoice":      {airMonitor, "a saved root is a cast change, and a recast is applied to the LIVE source"},\n'
assert old in s, "mZ5"
p.write_text(s.replace(old, "", 1))
