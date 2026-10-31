package app

import (
	"os"
	"strconv"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/synth"
)

// sweptWords is what the diagnostics say of a start's sweep: the speech
// processes a run killed outright left, ended, and any that could not be,
// which the next start tries again (I-3); "" when there were none.
func sweptWords(ended, failed int) string {
	switch {
	case ended == 0 && failed == 0:
		return ""
	case failed == 0:
		return "Speech: " + strconv.Itoa(ended) + " speech process(es) a previous run left were ended"
	}
	return "Speech: " + strconv.Itoa(ended) + " speech process(es) a previous run left were ended, " + strconv.Itoa(failed) + " could not be; the next start tries again"
}

// sayChildren readies the say children for this run (0.19.0 W9.2, FR-9.1,
// FR-9.2): it ends what a watchpost killed outright left behind, recorded in
// dir; records this run's children there; and ends them if the terminal
// hangs up. The stop it returns ends any still running, as the run exits.
// With no dir, nothing is swept or recorded and the children are still
// ended at exit. It says how many it swept, and how many it could not end.
func sayChildren(dir string, procs synth.Procs) (stop func(), swept, failed int) {
	if dir != "" {
		swept, failed = synth.SweepOrphans(dir, procs, os.Getuid(), time.Now())
	}
	synth.SetRecordDir(dir)
	unhook := endOnHangUp()
	return func() {
		unhook()
		synth.EndChildren()
	}, swept, failed
}
