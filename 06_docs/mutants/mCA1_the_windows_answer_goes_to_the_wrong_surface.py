import pathlib
# D-128. The Router forwards a window's KEY PRESSES to Observer while the window
# is composited over the console, and this deletes the other half: the REPLY to
# the command that window issued. The operator searches a location, presses
# enter, and the resolvedMsg is delivered to the console — which cannot read one
# and drops it. No result, no error, a window that looks like a dead key.
#
# Re-pointed 2026-09-14 (D-130): the debounce's two messages joined the case.
#
# THE MUTATION KEEPS THE FUNCTION so the build still uses it: observerScoped
# answers false for the search window's own reply, which is exactly the routing
# that shipped the defect.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """	case resolvedMsg, committedMsg, castSavedMsg, uiSavedMsg,
		locatePauseMsg, locateVerdictMsg:"""
new = """	case committedMsg, castSavedMsg, uiSavedMsg,
		locatePauseMsg, locateVerdictMsg:"""
assert old in s, "mCA1"
p.write_text(s.replace(old, new, 1))
