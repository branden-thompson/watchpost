import pathlib
# GO TO STANDBY stops the schedule and not the words. The Director takes the card
# off the air, the console says the station is off — and the read already in
# flight is a worker blocking on the engine, so the station talks to the end of
# the report with STANDBY on the screen.
p = pathlib.Path("app/mastercontrol.go"); s = p.read_text()
old = "\t\tm.silenceTheProgramme()\n\t\tm.run(func(m *mastercontrol) func() { return m.stopBed })"
new = "\t\tm.run(func(m *mastercontrol) func() { return m.stopBed })"
assert old in s, "mCE"
p.write_text(s.replace(old, new, 1))
