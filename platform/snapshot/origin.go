package snapshot

// origin.go — the application's compiled-in default location (T4.2, R-4).

// DefaultOrigin is Bonsall, CA: the origin the application ships with.
//
// SHOWN, NEVER USED SILENTLY (R-4, HUM LEAD 2026-09-01). It is displayed in
// Settings as the Default so a listener can see what the application would
// reason from, and Settings auto-opens on first run to prompt them for their
// own. Nothing reads it as a fence origin or a watchlist entry: a station that
// quietly scoped a listener's hazards to somebody else's town would be worse
// than one that admits it has nowhere to start.
//
// IT LIVES HERE, beside LocationRef, because it is a VALUE of that type and
// every surface that needs it already imports this package. It is a function
// rather than a package variable so it cannot be written (P10-06), and so a
// caller cannot hold a reference that another caller has since edited.
func DefaultOrigin() LocationRef {
	return LocationRef{
		Label: "Bonsall, CA",
		Tag:   "BNSL",
		Zip:   "92003",
		Lat:   33.2886,
		Lon:   -117.2247,
		TZ:    "America/Los_Angeles",
	}
}
