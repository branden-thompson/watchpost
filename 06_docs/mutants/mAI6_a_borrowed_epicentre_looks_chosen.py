import pathlib
# A borrowed epicentre stops saying it is borrowed, so the row shows the
# listener's default location as if the operator had chosen it — and it will move
# under them the next time they change their watchlist, with nothing having said
# so (D-115, HUM LEAD: "as long as we inform the user in some way").
p = pathlib.Path("modes/tty/setup_form.go"); s = p.read_text()
old = """		if d.cfg.Transmitter == nil {
			head += "  " + settingSupport("(following your default location)")
		}
"""
assert old in s, "mAI6"
p.write_text(s.replace(old, "", 1))
