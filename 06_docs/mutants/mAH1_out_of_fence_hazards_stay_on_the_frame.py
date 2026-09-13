import pathlib
# The projection returns out-of-fence cards again, so the console draws hazards
# the station will never broadcast — the carryover the HUM LEAD described:
# "alerts from Observer carry over" into a Broadcaster with a 25-mile fence. The
# air and the frame then name different cards, and the frame is the one the
# operator reads (D-114).
p = pathlib.Path("platform/lineup/projection.go"); s = p.read_text()
old = """		if c.OutOfFence {
			continue
		}
"""
assert old in s, "mAH1"
p.write_text(s.replace(old, "", 1))
