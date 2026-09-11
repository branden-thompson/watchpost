import pathlib
# A read that was HALTED reports itself finished. The operator went to standby, or
# a later read took the engine — and the schedule is told the card was read in
# full, so it leaves the line-up and is never offered again. DR-24 is that a read
# which ended early says so, always.
p = pathlib.Path("app/mainread.go"); s = p.read_text()
old = "\tcase st.State == player.Stopped || st.State == player.Failed:\n\t\tr.finish(false)"
new = "\tcase st.State == player.Stopped || st.State == player.Failed:\n\t\tr.finish(true)"
assert old in s, "mCC"
p.write_text(s.replace(old, new, 1))
