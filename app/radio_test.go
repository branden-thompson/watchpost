package app

import (
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestChooseNearestTakesTheFirstRelayedStation(t *testing.T) {
	// UAT 78/97: Synth is the default; [m] Nearest Relay takes the first
	// station the resolver lists with a mount — the covering transmitter
	// when relayed (the resolver puts it first), else the nearest relayed
	// one (Victorville for Oceanside). None → Synth.
	victorville := stream.Station{Transmitter: &stream.Transmitter{Callsign: "WXM66", Site: "Victorville"}, KM: 120, Covering: false, Mounts: []stream.Mount{{URL: "https://x/WXM66"}}}
	if st, live := chooseNearest([]stream.Station{victorville}, stream.LangEnglish); !live || st.Callsign != "WXM66" {
		t.Fatalf("Nearest Relay plays the nearest relayed station: %+v %v", st, live)
	}
	monterey := stream.Station{Transmitter: &stream.Transmitter{Callsign: "KEC49", Site: "Monterey"}, KM: 30, Covering: true, Mounts: []stream.Mount{{URL: "https://x/KEC49"}}}
	if st, live := chooseNearest([]stream.Station{monterey, victorville}, stream.LangEnglish); !live || st.Callsign != "KEC49" {
		t.Fatalf("the covering relayed transmitter comes first: %+v %v", st, live)
	}
	unrelayed := stream.Station{Transmitter: &stream.Transmitter{Callsign: "KEC62", Site: "San Diego"}, Covering: true}
	if _, live := chooseNearest([]stream.Station{unrelayed}, stream.LangEnglish); live {
		t.Fatal("no mount is no station")
	}
	if _, live := chooseNearest(nil, stream.LangEnglish); live {
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

func TestVoiceChipLabelIsTheChooserLabel(t *testing.T) {
	// UAT 91: the [V] chip shows "System Voice", not the spoken form.
	d := &radioDeck{}
	got := d.VoiceName()
	// THROUGH THE SEAM, not runtime.GOOS. VoiceName resolves the default via
	// the seam, so a test that branches on the real OS cannot be steered by
	// asPlatform and silently disagrees with the code the moment they differ.
	if runtimeGOOS == "darwin" {
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

// Quality pass Q1 (PR-9): a relay directory that stops answering is said
// once, in [S], and said again only after it has recovered.
func TestDirectoryOutageWarnsOncePerOutage(t *testing.T) {
	var warned []snapshot.Warning
	d := &radioDeck{warn: func(w snapshot.Warning) { warned = append(warned, w) }}
	down := []stream.Status{{Relay: "wxradio.org"}, {Relay: "weatherusa.net", Err: errors.New("tls: handshake failure"), Since: time.Now()}}
	d.noteDirectories(down)
	d.noteDirectories(down) // the next Tune, still down
	if len(warned) != 1 || warned[0].Code != snapshot.WarnRadioUnavailable || warned[0].Provider != "weatherusa.net" || !strings.Contains(warned[0].Message, "handshake") {
		t.Fatalf("one warning per outage, naming the relay and the reason: %+v", warned)
	}
	d.noteDirectories([]stream.Status{{Relay: "wxradio.org"}, {Relay: "weatherusa.net"}}) // recovered
	d.noteDirectories(down)
	if len(warned) != 2 {
		t.Fatalf("a new outage after recovery warns again, got %d", len(warned))
	}
	nilDeck := &radioDeck{}
	nilDeck.noteDirectories(down) // no hook (tests, report): never panics
}

// Quality pass Q1: the tune list spans every candidate station in order,
// and the label follows the mount that actually plays — a transmitter
// relayed only by a dead mount falls through to the next live station.
func TestTuneListSpansCandidatesAndLabelFollowsTheMount(t *testing.T) {
	a := stream.Station{Transmitter: &stream.Transmitter{Callsign: "KZZ41", Site: "Dead", FreqMHz: "162.400"}, KM: 10, Mounts: []stream.Mount{{Callsign: "KZZ41", URL: "http://wu/NWR/KZZ41.mp3", Relay: "weatherusa.net"}}}
	b := stream.Station{Transmitter: &stream.Transmitter{Callsign: "KEC80", Site: "Atlanta", FreqMHz: "162.550"}, KM: 40, Mounts: []stream.Mount{{Callsign: "KEC80", URL: "https://wx/GA-Atlanta-KEC80", Relay: "wxradio.org"}, {Callsign: "KEC80", URL: "http://wu/NWR/KEC80.mp3", Relay: "weatherusa.net"}}}
	urls, owners := tuneList([]stream.Station{a, b}, a)
	if len(urls) != 3 || urls[0] != a.Mounts[0].URL || urls[2] != b.Mounts[1].URL || owners[urls[1]].Callsign != "KEC80" {
		t.Fatalf("mounts flatten in station order with owners: %v %v", urls, owners)
	}
	d := &radioDeck{units: render.UnitF, mountOwner: owners}
	d.setMode("live", d.label(a), "weatherusa.net")
	d.followMount("")              // no mount yet: unchanged
	d.followMount("https://other") // not ours: unchanged
	d.followMount(a.Mounts[0].URL) // still the first station: unchanged
	if d.station != d.label(a) {
		t.Fatalf("label must stay on the first station until another mount plays, got %q", d.station)
	}
	d.followMount(b.Mounts[0].URL) // the engine fell through to Atlanta on wxradio
	if d.station != d.label(b) || d.detail != "wxradio.org" {
		t.Fatalf("label must follow the playing mount's station and relay, got %q / %q", d.station, d.detail)
	}
}

// THE WATCHLIST DWELL IS A VALUE, NOT A CONSTANT (HUM LEAD, UAT 2026-09-04).
//
// Waiting five minutes a station to see whether the rotation works is not a
// test anybody runs twice. The duration is overridable so the whole path —
// tick, dwell, tune, the next station taking the air — can be exercised in
// thirty seconds.
//
// A BAD VALUE IS THE DEFAULT, NOT AN ERROR. A mistyped variable that silently
// stopped the rotation would look exactly like the defect this exists to find.
func TestTheWatchlistDwellCanBeOverriddenForTesting(t *testing.T) {
	for _, tc := range []struct {
		env  string
		want time.Duration
	}{
		{"", liveDwell},
		{"30s", 30 * time.Second},
		{"2m", 2 * time.Minute},
		{"nonsense", liveDwell},
		{"0s", liveDwell},
		{"-30s", liveDwell},
	} {
		t.Run("WATCHPOST_WATCHLIST_DWELL="+tc.env, func(t *testing.T) {
			t.Setenv("WATCHPOST_WATCHLIST_DWELL", tc.env)
			if got := envWatchlistDwell(); got != tc.want {
				t.Errorf("envWatchlistDwell() = %v, want %v", got, tc.want)
			}
		})
	}
}

// THE LANGUAGE PREFERENCE DECIDES A TIE AND NOTHING ELSE.
//
// Coachella KIG78 and Coachella / Spanish WNG712 share a mast, so nothing about
// the geography prefers either and the resolver's order between them is
// arbitrary. Vista, CA got the Spanish feed that way, and the ruling was that
// the choice belongs to the listener (HUM LEAD, UAT 2026-09-04).
//
// The danger in a preference is that it stops being a tie-break: a listener who
// prefers Spanish must not be sent to a Spanish transmitter in another county
// over the English one covering theirs. Distance still decides; this only
// answers what distance leaves open.
func TestTheLanguagePreferenceOnlyBreaksATie(t *testing.T) {
	mounted := []stream.Mount{{URL: "http://example/1"}}
	tied := func(call, site string, km float64) stream.Station {
		return stream.Station{Transmitter: &stream.Transmitter{Callsign: call, Site: site}, KM: km, Mounts: mounted}
	}
	// The real pair, in the order the table's callsign tie-break yields.
	english := tied("KIG78", "Coachella", 127)
	spanish := tied("WNG712", "Coachella / Spanish", 127)

	for _, tc := range []struct {
		name     string
		stations []stream.Station
		prefer   string
		want     string
	}{
		{"a tie goes to the preference", []stream.Station{english, spanish}, stream.LangSpanish, "WNG712"},
		{"a tie goes to the preference from either order", []stream.Station{spanish, english}, stream.LangEnglish, "KIG78"},
		{"English is left alone when it already leads", []stream.Station{english, spanish}, stream.LangEnglish, "KIG78"},
		{"no preference keeps the resolver's order", []stream.Station{english, spanish}, "", "KIG78"},

		// THE HALF THAT MATTERS: a nearer station is not a tie.
		{"a nearer English station beats a farther Spanish one", []stream.Station{
			tied("KEC49", "Monterey", 40), spanish}, stream.LangSpanish, "KEC49"},
		{"a nearer Spanish station beats a farther English one", []stream.Station{
			tied("WNG652", "El Paso Spanish", 40), english}, stream.LangEnglish, "WNG652"},

		// Nor is a covering station, which outranks distance by design.
		{"a covering station is not displaced by a tie on distance", []stream.Station{
			{Transmitter: &stream.Transmitter{Callsign: "KEC62", Site: "San Diego"}, KM: 127, Covering: true, Mounts: mounted},
			spanish}, stream.LangSpanish, "KEC62"},

		// An unrelayed station cannot win the tie it appears to be in.
		{"an unrelayed match does not win", []stream.Station{english,
			{Transmitter: &stream.Transmitter{Callsign: "WNG712", Site: "Coachella / Spanish"}, KM: 127}}, stream.LangSpanish, "KIG78"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, live := chooseNearest(tc.stations, tc.prefer)
			if !live {
				t.Fatal("a relayed station must be chosen")
			}
			if st.Callsign != tc.want {
				t.Errorf("chose %s (%s), want %s", st.Callsign, st.Site, tc.want)
			}
		})
	}
}

// THE CHOSEN STATION LEADS THE TUNE LIST.
//
// The engine starts at urls[0] and the deck is labelled with the station
// chooseNearest picked. Those agreed for as long as chooseNearest meant "the
// first candidate with a mount"; the language preference (HUM LEAD, UAT
// 2026-09-04) may pick a co-located station further down, and then the deck
// would name one transmitter while the audio came from the one beside it.
//
// Every other candidate still follows, in order and without duplication, so the
// fall-through a dead mount depends on is unchanged.
func TestTheChosenStationLeadsTheTuneList(t *testing.T) {
	mount := func(call, url string) []stream.Mount {
		return []stream.Mount{{Callsign: call, URL: url, Relay: "weatherusa.net"}}
	}
	english := stream.Station{Transmitter: &stream.Transmitter{Callsign: "KIG78", Site: "Coachella"},
		KM: 127, Mounts: mount("KIG78", "http://wu/NWR/KIG78_2.mp3")}
	spanish := stream.Station{Transmitter: &stream.Transmitter{Callsign: "WNG712", Site: "Coachella / Spanish"},
		KM: 127, Mounts: mount("WNG712", "http://wu/NWR/WNG712.mp3")}
	far := stream.Station{Transmitter: &stream.Transmitter{Callsign: "KWO37", Site: "Los Angeles"},
		KM: 150, Mounts: mount("KWO37", "http://wu/NWR/KWO37.mp3")}
	order := []stream.Station{english, spanish, far}

	// A listener who prefers Spanish gets WNG712 chosen — and must HEAR it.
	urls, owners := tuneList(order, spanish)
	if urls[0] != spanish.Mounts[0].URL {
		t.Errorf("the chosen station leads, got %s", urls[0])
	}
	if owners[urls[0]].Callsign != "WNG712" {
		t.Errorf("the leading mount belongs to the chosen station, got %s", owners[urls[0]].Callsign)
	}
	// Every candidate is still reachable, exactly once, so a dead mount still
	// falls through to the next live station.
	if len(urls) != 3 {
		t.Fatalf("every mount appears once: %v", urls)
	}
	seen := map[string]bool{}
	for _, u := range urls {
		if seen[u] {
			t.Errorf("%s appears twice", u)
		}
		seen[u] = true
	}

	// And the ordinary case is unchanged: the resolver's first is the choice.
	urls, _ = tuneList(order, english)
	if urls[0] != english.Mounts[0].URL || len(urls) != 3 {
		t.Errorf("the usual order is untouched: %v", urls)
	}
}

// THE FAULT WINDOW IS OFFERED WHAT HAS NOT BEEN TRIED.
//
// Offering the listener the mount that just went silent is offering them the
// fault again, and it is the one candidate guaranteed not to work. The rest
// arrive in the engine's own fall-through order, so "Recommended" is what it
// would have reached next anyway (MVS-D-76).
func TestASilentRelayOffersTheStationsNotYetTried(t *testing.T) {
	mount := func(call, url string) []stream.Mount {
		return []stream.Mount{{Callsign: call, URL: url, Relay: "weatherusa.net"}}
	}
	dead := stream.Station{Transmitter: &stream.Transmitter{Callsign: "KIG78", Site: "Coachella", FreqMHz: "162.400"},
		KM: 127, Mounts: mount("KIG78", "http://wu/KIG78.mp3")}
	next := stream.Station{Transmitter: &stream.Transmitter{Callsign: "WNG712", Site: "Coachella / Spanish", FreqMHz: "162.525"},
		KM: 127, Mounts: mount("WNG712", "http://wu/WNG712.mp3")}
	// TWO MOUNTS, like the real KIH62: wxradio.org and weatherusa.net both
	// carry it. A station relayed twice must be OFFERED once — a list naming it
	// twice reads as two different options that do the same thing.
	far := stream.Station{Transmitter: &stream.Transmitter{Callsign: "KWO37", Site: "Los Angeles", FreqMHz: "162.550"},
		KM: 150, Mounts: []stream.Mount{
			{Callsign: "KWO37", URL: "https://wx/CA-LA-KWO37", Relay: "wxradio.org"},
			{Callsign: "KWO37", URL: "http://wu/KWO37.mp3", Relay: "weatherusa.net"},
		}}

	urls, owners := tuneList([]stream.Station{dead, next, far}, dead)
	d := &radioDeck{units: render.UnitF, mountOwner: owners, mountURLs: urls}

	got := d.silentCandidates("http://wu/KIG78.mp3")
	if len(got) != 2 {
		t.Fatalf("the two untried STATIONS are offered — not their three mounts — got %d: %v", len(got), got)
	}
	if got[0].Key != "WNG712" || got[1].Key != "KWO37" {
		t.Errorf("they keep the engine's fall-through order, got %s then %s", got[0].Key, got[1].Key)
	}
	for _, c := range got {
		if c.Key == "KIG78" {
			t.Error("the silent station must never be offered back")
		}
		if c.Label == "" {
			t.Error("every candidate reads as something a listener can choose")
		}
	}

	// A mount from a tune that has already been replaced offers nothing rather
	// than offering everything: the window would be about a station that is no
	// longer playing.
	if got := d.silentCandidates("http://wu/gone.mp3"); got != nil {
		t.Errorf("an unknown mount offers nothing, got %v", got)
	}
}
