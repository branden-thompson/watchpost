import pathlib
# [k] drops the card outright with no question asked. It is the DESTRUCTIVE one
# (FR-3.7) — "a confirm guards the accidental keypress" — and the card is gone to
# the discard pile on a single stray press (D-118).
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """	case "k":
		r.observer.cardAct, r.observer.cardErr = cardActionDrop, ""
		return r, true"""
assert old in s, "mAL3"
new = """	case "k":
		if drop := r.observer.cfg.DropCard; drop != nil {
			drop(r.observer.cardID)
		}
		r.observer = r.observer.close()
		return r, true"""
p.write_text(s.replace(old, new, 1))
