package cast

// emergency_mute_test.go — leave-now cannot be silenced.
//
// `emergency_tone_test.go` STATES THE RULE AND ENFORCES HALF OF IT: ClassEmergency
// is excluded from `Classes()`, so no Setup row can mute it. The other half is
// `Muted`, which answers TRUE for every class when the mode is mute and the list
// is empty — Emergency included — and `Config.withToneCompat` migrates a
// pre-0.14.0 `ticker_muted = true` into exactly that state.
//
// SO AN UPGRADING LISTENER LOST THE ATTENTION TONE SILENTLY. The words still
// read, so the alert is not lost; what is lost is the three-repeat tone that
// tells someone who is NOT looking to leave before any word is spoken. And
// because `toggleClass` materialises the six listed keys, the bad state is
// sticky and invisible — there is no row to un-tick.
//
// THIS IS P-9's SHAPE: a member of a closed set inheriting a rule ratified
// before it existed.

import "testing"

func TestTheLeaveNowToneIsNeverMuted(t *testing.T) {
	for _, tc := range []struct {
		name string
		t    Tones
	}{
		{"mute with nothing ticked (the migration's state)", Tones{Mode: ModeTonesMute}},
		{"mute with every listed class ticked", func() Tones {
			out := Tones{Mode: ModeTonesMute}
			for _, c := range Classes() {
				out.Muted = append(out.Muted, c.Key())
			}
			return out
		}()},
		{"mute naming emergency outright", Tones{Mode: ModeTonesMute, Muted: []string{ClassEmergency.Key()}}},
	} {
		if Muted(ClassEmergency, tc.t) {
			t.Errorf("%s: the leave-now tone is silenced — a listener who is not looking gets "+
				"no attention signal before the words begin (#18)", tc.name)
		}
	}
}

// AND EVERY OTHER CLASS IS STILL MUTABLE, or the fix would have taken the
// setting away from the listener rather than protecting one tone.
func TestTheOtherClassesAreStillSilenceable(t *testing.T) {
	empty := Tones{Mode: ModeTonesMute}
	for _, c := range Classes() {
		if !Muted(c, empty) {
			t.Errorf("%s is not silenced by an empty mute list; the listener asked for silence", c.Key())
		}
	}
}
