import pathlib
# Observer's repeat mode reaches the LIVE source again — and `d.source` during a
# main-track read IS the Broadcaster's card (BD-9). Reached from the Settings
# window the console forwards to, `RepeatOne` sets the card ON THE AIR to loop: it
# reads for ever, never finishes, and the line-up never advances. This is the
# defect the HUM LEAD's air ruling was measured on (D-91, F-100).
p = pathlib.Path("app/radio_queue.go"); s = p.read_text()
old = "\tif src != nil && d.monitorHasTheAir() {"
new = "\tif src != nil {"
assert old in s, "mZ1"
p.write_text(s.replace(old, new, 1))
