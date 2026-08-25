package app

import (
	"runtime"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestChooseNearestTakesTheFirstRelayedStation(t *testing.T) {
	// UAT 78/97: Synth is the default; [m] Nearest Relay takes the first
	// station the resolver lists with a mount — the covering transmitter
	// when relayed (the resolver puts it first), else the nearest relayed
	// one (Victorville for Oceanside). None → Synth.
	victorville := stream.Station{Transmitter: &stream.Transmitter{Callsign: "WXM66", Site: "Victorville"}, KM: 120, Covering: false, Mounts: []stream.Mount{{URL: "https://x/WXM66"}}}
	if st, live := chooseNearest([]stream.Station{victorville}); !live || st.Callsign != "WXM66" {
		t.Fatalf("Nearest Relay plays the nearest relayed station: %+v %v", st, live)
	}
	monterey := stream.Station{Transmitter: &stream.Transmitter{Callsign: "KEC49", Site: "Monterey"}, KM: 30, Covering: true, Mounts: []stream.Mount{{URL: "https://x/KEC49"}}}
	if st, live := chooseNearest([]stream.Station{monterey, victorville}); !live || st.Callsign != "KEC49" {
		t.Fatalf("the covering relayed transmitter comes first: %+v %v", st, live)
	}
	unrelayed := stream.Station{Transmitter: &stream.Transmitter{Callsign: "KEC62", Site: "San Diego"}, Covering: true}
	if _, live := chooseNearest([]stream.Station{unrelayed}); live {
		t.Fatal("no mount is no station")
	}
	if _, live := chooseNearest(nil); live {
		t.Fatal("no stations: Synth")
	}
}

func TestParseSayVoicesIsTheCuratedListInOrder(t *testing.T) {
	// UAT 87: only the curated radio voices, in HUM LEAD's order, and only
	// when installed; Eddy/Reed count only in their English (US) variants.
	listing := "Alex                en_US    # Most people recognize me by my voice.\n" +
		"Tessa               en_ZA    # Hello! My name is Tessa.\n" +
		"Eddy (English (UK)) en_GB    # Hello! My name is Eddy.\n" +
		"Eddy (English (US)) en_US    # Hello! My name is Eddy.\n" +
		"Daniel              en_GB    # Hello! My name is Daniel.\n" +
		"Samantha            en_US    # Hello! My name is Samantha.\n" +
		"Aman (English (India)) en_IN    # Hello! My name is Aman.\n" +
		"Aman (English (India)) en_IN    # Hi, I’m Siri!\n"
	got := parseSayVoices(listing)
	want := []string{systemVoice, "Aman (English (India))", "Daniel", "Eddy (English (US))", "Samantha", "Tessa"} // UAT 88: system voice first, Samantha back
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("voices = %v", got)
	}
	if len(macVoices()) != 11 {
		t.Fatal("system voice + ten curated voices")
	}
}

func TestNextInQueueWrapsAndStartsAtTheTop(t *testing.T) {
	// UAT 93: Watchlist advances by key, wraps at the end, starts at the top
	// when the current location is not a favourite, and has nowhere to go
	// on an empty queue.
	q := []snapshot.LocationRef{{Label: "A", Lat: 1, Lon: 1}, {Label: "B", Lat: 2, Lon: 2}, {Label: "C", Lat: 3, Lon: 3}}
	if n, ok := nextInQueue(q, q[0]); !ok || n.Label != "B" {
		t.Fatalf("after A comes B, got %v %v", n, ok)
	}
	if n, ok := nextInQueue(q, q[2]); !ok || n.Label != "A" {
		t.Fatalf("after C wraps to A, got %v %v", n, ok)
	}
	if n, ok := nextInQueue(q, snapshot.LocationRef{Label: "recent", Lat: 9, Lon: 9}); !ok || n.Label != "A" {
		t.Fatalf("a non-favourite starts at the top, got %v %v", n, ok)
	}
	if _, ok := nextInQueue(nil, q[0]); ok {
		t.Fatal("an empty queue has nowhere to go")
	}
}

func TestVoiceChipLabelIsTheChooserLabel(t *testing.T) {
	// UAT 91: the [V] chip shows "System Voice", not the spoken form.
	d := &radioDeck{}
	got := d.VoiceName()
	if runtime.GOOS == "darwin" {
		if got != systemVoice && got != defaultMacVoice {
			t.Fatalf("chip label = %q", got)
		}
	} else if got != "" { // no Piper in a fresh test dir: the chip shows "—", never a Mac voice name (Linux F3)
		t.Fatalf("chip label before a voice is installed = %q, want empty", got)
	}
	d.voiceID = "Karen"
	if d.VoiceName() != "Karen" {
		t.Fatal("chosen voice labels the chip")
	}
}

func TestPiperVoiceChooserListsTheCatalogue(t *testing.T) {
	// UAT 118: on Linux/Windows the [V] chooser lists every curated voice,
	// installed or not — a pick downloads on first use; the chip reads the
	// installed voice (empty before any install, Linux F3), and the chosen
	// name resolves to its catalogue entry.
	if runtime.GOOS == "darwin" {
		t.Skip("macOS lists `say -v ?` voices")
	}
	d := &radioDeck{voiceDir: t.TempDir()}
	names := d.discoverVoices()
	if len(names) != 6 || names[0] != "Lessac" || names[1] != "Amy" {
		t.Fatalf("catalogue in the chooser: %v", names)
	}
	if d.defaultVoice() != "" {
		t.Fatal("nothing installed: no default name yet")
	}
	d.voiceID = "Alan"
	if d.piperSpec().Key != "en_GB-alan-medium" {
		t.Fatal("the pick resolves to its catalogue voice")
	}
	d.voiceID = "not-a-voice"
	if d.piperSpec().Key != "en_US-lessac-medium" {
		t.Fatal("an unknown pick falls back to the default")
	}
}
