import pathlib
# The air returns to the monitor and the programme keeps reading. "If I'm in the
# Broadcaster UI, then the Observer Mode must be Silent" (HUM LEAD, 2026-09-10)
# has a matching half he ruled in the same breath: coming back is "like if
# Observer was first opened", and Observer first opened is silent.
p = pathlib.Path("app/mastercontrol.go"); s = p.read_text()
old = "\tif to != lineup.AirProgramme {\n\t\tm.silenceTheProgramme()\n\t}\n"
new = ""
assert old in s, "mCF"
p.write_text(s.replace(old, new, 1))
