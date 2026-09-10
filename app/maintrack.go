package app

import "os"

// maintrack.go — THE MERGE'S SWITCH, AND IT IS TEMPORARY (0.16.0 P3).
//
// P3 replaces the rotation's own audio ownership with the schedule's. Two
// owners become one, and the window in which both could speak is the defect
// the whole batch exists to close — so the change lands in stages, and this
// file is the stage marker. IT IS DELETED AT P3(d), together with startSynth's
// direct path; a switch that outlives the merge would be a second way for the
// station to behave, which is the thing being removed.
//
// THE DEFAULT IS TODAY'S BEHAVIOUR. A batch that is mid-flight must not change
// what a listener hears, and "off" is what makes every commit before the flip
// a no-op for anyone not deliberately looking.

// mainTrackStage is how far the merge is switched on for this process.
type mainTrackStage int

const (
	// mainTrackOff is today's station: the rotation owns its own audio and the
	// Director is never told a location needs a read. The default.
	mainTrackOff mainTrackStage = iota

	// mainTrackDark reports the need, queues the card, composes its words and
	// publishes it to the console — and STOPS SHORT OF THE AIR. The producer
	// runs and its decisions can be compared against the live path's, which is
	// the observation the plan requires before the merge owns the air. The
	// rotation still speaks through its own path, so the station sounds
	// unchanged; the cost is that a report is composed twice while dark.
	mainTrackDark

	// numMainTrackStages bounds the registry; it is not itself a stage.
	numMainTrackStages
)

// mainTrack reads the stage from the environment, once per ask.
//
// AN UNRECOGNISED VALUE IS OFF: a typo in a shell profile must not silently
// change what a station does.
//
// "live" IS GONE (D-33, 2026-09-09). It meant "the card is read through the
// arbiter", and the arbiter does not read the programme — a chosen read
// REPLACES the bed. The stage that expressed the wrong design went with it.
func mainTrack() mainTrackStage {
	switch os.Getenv("WATCHPOST_MAINTRACK") {
	case "dark":
		return mainTrackDark
	}
	return mainTrackOff
}

// stageNames is the registry, indexed by mainTrackStage. DERIVED FROM IT, so
// the diagnostic and the switch cannot name different things (INST-1).
func stageNames() [numMainTrackStages]string {
	return [numMainTrackStages]string{
		mainTrackOff:  "off",
		mainTrackDark: "dark",
	}
}

// String names the stage for the dark run's log. An undeclared stage names
// itself as such rather than as one of the three, because a diagnostic that
// silently reports "off" for a corrupt value is worse than one that says it
// does not know.
func (s mainTrackStage) String() string {
	if s < 0 || s >= numMainTrackStages {
		return "undeclared"
	}
	return stageNames()[s]
}

// reports says whether the deck tells the Director that a location needs a
// read. Dark and live both do; that is what makes dark an observation of the
// real producer rather than a simulation of one.
func (s mainTrackStage) reports() bool { return s == mainTrackDark }
