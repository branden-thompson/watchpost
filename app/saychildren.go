package app

import (
	"os"

	"github.com/branden-thompson/watchpost/domains/radio/synth"
)

// sayChildren readies the say children for this run (0.19.0 W9.2, FR-9.1,
// FR-9.2): it ends what a watchpost killed outright left behind, recorded in
// dir; records this run's children there; and ends them if the terminal
// hangs up. The stop it returns ends any still running, as the run exits.
// With no dir, nothing is swept or recorded and the children are still
// ended at exit. It says how many it swept.
func sayChildren(dir string, procs synth.Procs) (stop func(), swept int) {
	if dir != "" {
		swept = synth.SweepOrphans(dir, procs, os.Getuid())
	}
	synth.SetRecordDir(dir)
	unhook := endOnHangUp()
	return func() {
		unhook()
		synth.EndChildren()
	}, swept
}
