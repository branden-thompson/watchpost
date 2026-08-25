# AI-11 — go-studs provenance and distribution (RS-1)

Question at DISCOVER: could a public Watchpost depend on go-studs, the author's terminal UI kit,
which at the time lived in a private repository under a module path the public could not
resolve (and `go install` ignores `replace` directives regardless)?

Facts established: go-studs carries an MIT LICENSE in the author's own name; the packages
Watchpost uses (rendering, components, theme, tokens) have no dependency on any private service;
the risk was purely one of *reachability* of the module path.

Resolution (BUILD exit, 2026-08-25, D-25): the packages are carried inside this module under
`third_party/go-studs` (with their LICENSE and a NOTICE naming the upstream commit), refreshed
by `scripts/sync-go-studs.sh` from the author's checkout. No private access is needed to build,
`go install` works, and RS-1 is closed. The original access analysis is superseded by this note.
