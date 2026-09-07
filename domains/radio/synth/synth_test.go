package synth

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/branden-thompson/watchpost/platform/render"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func f64(v float64) *float64 { return &v }

func TestComposeReadsLikeNWR(t *testing.T) {
	loc := snapshot.Location{Label: "Oceanside, CA", Harmonized: snapshot.Conditions{
		Condition: "partly_cloudy", Temp: f64(22.8), HumidityPct: f64(66), Wind: f64(4), WindDirDeg: f64(250),
		Source: snapshot.SourceInfo{Provider: "nws"}},
		Alerts: []snapshot.Alert{{ID: "a1", Headline: "Heat Advisory issued August 24 at 208PM PDT until 8 PM PDT Friday by NWS San Diego CA", Description: "* WHAT...Hot.\n\n* WHERE...Coast."}}}
	now := time.Date(2026, 8, 24, 16, 5, 0, 0, time.UTC)
	segs := std.Compose(loc, []Product{{ID: "p1", Type: "ZFP", Text: ".TONIGHT...Mostly clear. Lows 66 to 69.\n\n$$"}}, now, true, "Samantha", Station{Callsign: "KEC62", Site: "San Diego", State: "CA", FreqMHz: "162.400"}, Reports{}, render.Clock12)
	texts := make([]string, 0, len(segs))
	for _, s := range segs {
		texts = append(texts, s.Text)
	}
	joined := strings.Join(texts, "\n")
	for _, want := range []string{
		"This is Watchpost Weather Radio serving Oceanside, California. A version of this forecast is also broadcast live from KEC62, San Diego, California broadcasting on 162.400 MHz and is accessible via NOAA radio devices and receivers. Watchpost Weather Radio forecasts may be delayed and are not intended for life safety use.", // UAT 112 script; a 2 s pause follows (112.3)
		"This forecast is from the National Oceanic and Atmospheric Administration and is for Monday, August 24 until Sunday, August 30.",
		"Current conditions: partly cloudy, temperature 73 degrees, humidity 66 percent, wind west at 9 miles per hour.",
		"Heat Advisory issued August 24 at 2:08 PM Pacific Daylight Time until 8 PM Pacific Daylight Time Friday by National Weather Service San Diego California. What: Hot. Where: Coast.", // UAT 82: headlines follow the word rules too; UAT 95: bullet labels are statements ("What:"), never "What?"
		"Tonight. Mostly clear. Lows 66 to 69.",
		"This is Samantha for Watchpost Weather Radio. You can change your correspondents in Settings."} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q:\n%s", want, joined)
		}
	}
	if !strings.HasPrefix(segs[3].Key, "alert:a1:") || !strings.HasPrefix(segs[4].Key, "ZFP:p1:") || segs[len(segs)-1].Key != "tail:"+VoiceToken {
		t.Fatalf("segment keys carry issuance identity: %v", []string{segs[3].Key, segs[4].Key, segs[len(segs)-1].Key})
	}
	// Composer.Tail no longer substitutes "your correspondent" for an empty
	// name — spokenName owns that, once, where the voice is resolved (RS-18).
	// Tail is handed a name that is already right.
	if got := std.Tail("your correspondent"); !strings.Contains(got, "This is your correspondent for Watchpost Weather Radio.") {
		t.Fatalf("tail with the resolved fallback name: %q", got)
	}
	// Each section is tagged with the role that will read it (FR-3).
	for i, want := range []cast.Role{cast.Station, cast.Station, cast.Weather, cast.Weather, cast.Weather, cast.Station} {
		if segs[i].Role != want {
			t.Errorf("segment %d (%s) has role %v, want %v", i, segs[i].Key, segs[i].Role, want)
		}
	}
	if (PiperVoice{Install: Install{Model: "/x/en_US-lessac-medium.onnx"}}).Name() != "Lessac" {
		t.Fatal("Piper voices are named by their given name")
	}
}

// toneVoice renders each text as N ms of a constant sample (test double).
type toneVoice struct{ calls atomic.Int32 }

func (v *toneVoice) Name() string { return "tone" }
func (v *toneVoice) Rate() int    { return 22050 }
func (v *toneVoice) Say(_ context.Context, text string) ([]byte, error) {
	v.calls.Add(1)
	n := 22050 / 10 // 100 ms
	out := make([]byte, n*2)
	for i := range n {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(int16(len(text))))
	}
	return out, nil
}

func TestSourceCyclesRendersOnceAndWidensToStereo(t *testing.T) {
	v := &toneVoice{}
	var cycles atomic.Int32
	src, err := NewSource(v, func(context.Context) ([]Segment, error) {
		cycles.Add(1)
		return []Segment{{Key: "a", Text: "one"}, {Key: "b", Text: "two two"}}, nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	src.gap = 10 * time.Millisecond
	src.Loop(true)
	ctx, cancel := context.WithCancel(context.Background())
	r := src.Open(ctx)
	// One segment = 100 ms mono -> 2205 frames stereo = 8820 bytes, plus a 10 ms gap.
	buf := make([]byte, 8820)
	if _, err := io.ReadFull(r, buf); err != nil {
		t.Fatal(err)
	}
	if l, rr := binary.LittleEndian.Uint16(buf[0:]), binary.LittleEndian.Uint16(buf[2:]); l != 3 || rr != 3 {
		t.Fatalf("stereo widening must duplicate the sample: %d %d", l, rr)
	}
	// Read through the second cycle: the same keys must not re-render.
	rest := make([]byte, 4*(8820+22050*4/100))
	_, _ = io.ReadFull(r, rest)
	cancel()
	if cycles.Load() < 2 {
		t.Fatalf("source must ask for the next cycle, got %d", cycles.Load())
	}
	if v.calls.Load() != 2 {
		t.Fatalf("segments render once and are cached by key, got %d renders", v.calls.Load())
	}
}

func TestSourceStopsWhenTheVoiceIsGone(t *testing.T) {
	src, _ := NewSource(TextTicker{}, func(context.Context) ([]Segment, error) { return []Segment{{Key: "a", Text: "x"}}, nil }, nil)
	r := src.Open(context.Background())
	if _, err := io.ReadAll(r); err != nil && !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
}

func TestProductsLatestPerTypeInBroadcastOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/products/types/ZFP/"):
			b, _ := os.ReadFile("testdata/zfp_list.json")
			_, _ = w.Write(b)
		case strings.HasPrefix(r.URL.Path, "/products/types/"):
			_, _ = w.Write([]byte(`{"@graph":[]}`)) // HWO/SPS/NOW not issued
		case strings.HasPrefix(r.URL.Path, "/products/"):
			b, _ := os.ReadFile("testdata/zfp_sgx.json")
			_, _ = w.Write(b)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
	ps, err := NewProducts(c, srv.URL).Latest(context.Background(), "SGX")
	if err != nil || len(ps) != 1 || ps[0].Type != "ZFP" || ps[0].Office != "SGX" || !strings.Contains(ps[0].Text, "Zone Forecast Product") {
		t.Fatalf("products = %+v, err %v", ps, err)
	}
	if _, err := NewProducts(c, srv.URL).Latest(context.Background(), ""); err == nil {
		t.Fatal("office is required")
	}
}

func TestWAVPCMAndSafeJoin(t *testing.T) {
	wav := append([]byte("RIFF\x00\x00\x00\x00WAVEfmt \x02\x00\x00\x00\x01\x00data\x04\x00\x00\x00"), 1, 2, 3, 4)
	pcm, err := wavPCM(wav)
	if err != nil || len(pcm) != 4 || pcm[0] != 1 {
		t.Fatalf("wavPCM: %v %v", pcm, err)
	}
	if _, err := wavPCM([]byte("nope")); err == nil {
		t.Fatal("non-WAVE must be refused")
	}
	if _, err := safeJoin("/tmp/x", "../../etc/passwd"); err == nil {
		t.Fatal("archive entries must not escape the install dir")
	}
	if !PiperSupported() && runtime.GOOS != "darwin" {
		t.Fatalf("manifest must cover %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

func TestSayVoiceOnDarwinNarratesHostileTextSafely(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS built-in voice")
	}
	if _, err := os.Stat("/usr/bin/say"); err != nil {
		t.Skip("no say binary")
	}
	// §10.5: shell metacharacters in product text are inert — they are
	// narrated from a file, never interpreted.
	//
	// Bounded because this shells out to the real /usr/bin/say, which can wedge
	// (seen: a 10-minute suite panic on a loaded box, 0.14.0 P1). One sentence
	// renders in well under a second; the bound turns a wedge into a legible
	// failure of THIS test instead of a timeout panic that takes the package
	// with it.
	//
	// TWO MINUTES, NOT THIRTY SECONDS. This is a FAILURE bound, not an assertion
	// about speed, so it costs nothing on a healthy run and only decides how
	// much load it tolerates before calling a wedge. At 30 s it fired during a
	// full `make race` — the whole suite under the race detector, with `say` a
	// real process competing for the box — and reported "signal: killed" for a
	// test that passes 3/3 in isolation. A gate that fails for load teaches
	// people to re-run it rather than read it, which is how a real failure gets
	// waved through (the same argument as F-29).
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// A CONTROL RENDER FIRST, to tell "this box cannot speak" apart from "our
	// input broke it". GitHub's macOS runner has /usr/bin/say and produced 236
	// bytes for a whole sentence — it is present and not functional. Failing
	// there says nothing about the security property this test exists for, and a
	// gate that fails for the environment teaches people to re-run it.
	benign, err := SayVoice{}.Say(ctx, "One sentence of ordinary text.")
	if err != nil || len(benign) < 22050 {
		t.Skipf("say is present but not rendering audio on this host (%d bytes, err=%v): the shell-safety property cannot be observed here", len(benign), err)
	}

	pcm, err := SayVoice{}.Say(ctx, "Test $(echo pwned) `id` ; rm -rf / ok.")
	if err != nil {
		t.Fatalf("say did not render within the bound (or failed): %v", err)
	}
	// Against the CONTROL, not an absolute: hostile text must render like
	// ordinary text. If it renders far shorter, something ate part of it.
	if len(pcm) < len(benign)/2 {
		t.Fatalf("hostile text rendered %d bytes against %d for ordinary text — it was not narrated intact", len(pcm), len(benign))
	}
}

func TestFilterUGCKeepsOnlyTheLocationsBlocks(t *testing.T) {
	raw, _ := os.ReadFile("testdata/zfp_sgx.json")
	var doc struct {
		ProductText string `json:"productText"`
	}
	_ = json.Unmarshal(raw, &doc)
	got := FilterUGC(doc.ProductText, "CAZ554", "CAC059")
	if !strings.Contains(got, "Zone Forecast Product for Extreme Southwest California") {
		t.Fatal("preamble is kept")
	}
	if !strings.Contains(got, "Orange County Inland Areas") || strings.Contains(got, "Orange County Coastal Areas") {
		t.Fatalf("only the location's zone block is kept:\n%s", got)
	}
	if FilterUGC("no ugc here", "CAZ554", "") != "no ugc here" {
		t.Fatal("products without UGC lines pass through")
	}
	codes := expandUGC("CAZ043>045-050-CAC073")
	for _, want := range []string{"CAZ043", "CAZ044", "CAZ045", "CAZ050", "CAC073"} {
		if !codes[want] {
			t.Fatalf("UGC ranges expand: missing %s in %v", want, codes)
		}
	}
	if codes["CAZ046"] {
		t.Fatal("range must not over-expand")
	}
}

func TestNarrationRulesUAT81(t *testing.T) {
	paras := Normalize("Special Weather Statement\nNational Weather Service San Diego CA\n442 PM PDT Mon Aug 24 2026\n\nCA-San Diego County Mountains-\n1223 PM PDT Mon Aug 24 2026\n\nAt 442 PM PDT, gusty winds were reported near Julian.\n\nLAT...LON 3300 11700 3310 11720\n     3320 11730\n\nTIME...MOT...LOC 2342Z 260DEG 15KT 3305 11710\n\n$$")
	joined := strings.Join(paras, "\n")
	for _, want := range []string{"San Diego California", "California-San Diego County Mountains", "At 4:42 PM Pacific Daylight Time"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q:\n%s", want, joined)
		}
	}
	for _, bad := range []string{"LAT", "3300", "MOT", "2342"} {
		if strings.Contains(joined, bad) {
			t.Fatalf("polygon coordinates must not be narrated (%q):\n%s", bad, joined)
		}
	}
	long := strings.Repeat("Highs 80 to 84 at the beaches to 86 to 91 farther inland. ", 12)
	segs := Segments([]string{strings.TrimSpace(long)})
	if len(segs) < 3 {
		t.Fatalf("long paragraphs split into segments, got %d", len(segs))
	}
	for _, sg := range segs {
		if len(sg) > maxSegmentChars || !strings.HasSuffix(sg, ".") {
			t.Fatalf("segment must end at a sentence and stay short: %q", sg)
		}
	}
	if got := ExpandStates("HEAT ADVISORY IN EFFECT. Visit OR call. Oceanside, CA and CA-San Diego and San Diego CA and I am OK"); got != "HEAT ADVISORY IN EFFECT. Visit OR call. Oceanside, California and California-San Diego and San Diego California and I am OK" {
		t.Fatalf("state expansion needs place context: %q", got)
	}
	if std.Lead("Oceanside, CA", Station{}, time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC), render.Clock12) != "This is Watchpost Weather Radio serving Oceanside, California. Watchpost Weather Radio forecasts may be delayed and are not intended for life safety use. This forecast is from the National Oceanic and Atmospheric Administration and is for Monday, August 24 until Sunday, August 30." { // no known station: the live sentence is left out (UAT 112)
		t.Fatal("the lead expands the state")
	}
}

// heldVoice speaks its first segment normally and then BLOCKS inside the second
// until the context is cancelled, so a cancellation lands while a write is
// genuinely pending — which is the state UAT 81 is about.
//
// The previous fake could not pose that state. toneVoice returns 100 ms of audio
// instantly whatever the text, so both segments were produced immediately and
// the reader could drain to a clean EOF before the cancellation was observed;
// then io.ReadAll returned a nil error and the test failed for being right too
// early. That is a SUCCESS bound in disguise, and it failed once in ten on CI
// (F-50).
type heldVoice struct {
	calls   atomic.Int32
	entered chan struct{}
	once    sync.Once
}

func (v *heldVoice) Name() string { return "held" }
func (v *heldVoice) Rate() int    { return 22050 }

func (v *heldVoice) Say(ctx context.Context, text string) ([]byte, error) {
	if v.calls.Add(1) == 1 {
		n := 22050 / 10 // 100 ms, as toneVoice
		out := make([]byte, n*2)
		for i := range n {
			binary.LittleEndian.PutUint16(out[i*2:], uint16(int16(len(text))))
		}
		return out, nil
	}
	v.once.Do(func() { close(v.entered) })
	<-ctx.Done() // held open until the test cancels
	return nil, ctx.Err()
}

func TestSourceStopsFastWhileMidSegment(t *testing.T) {
	// UAT 81: cancelling mid-segment unblocks the pending write at once.
	v := &heldVoice{entered: make(chan struct{})}
	src, _ := NewSource(v, func(context.Context) ([]Segment, error) {
		return []Segment{{Key: "a", Text: strings.Repeat("x", 2000)}, {Key: "b", Text: "y"}}, nil
	}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	r := src.Open(ctx)
	buf := make([]byte, 1024)
	if _, err := r.Read(buf); err != nil {
		t.Fatal(err)
	}

	// THE PREMISE, ASSERTED RATHER THAN ASSUMED: the source must actually be
	// mid-segment. If it never reaches the second one there is no pending write
	// to unblock, and a pass would mean nothing.
	select {
	case <-v.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the source never began the second segment: this test cannot pose the cancellation it is about")
	}

	// THE PROPERTY IS PROMPTNESS, NOT THE ERROR. Requiring io.ReadAll to report a
	// non-nil error is not stable: after a cancellation the stream may equally
	// end cleanly, and CI showed both outcomes on macOS within one commit. What
	// UAT 81 is actually about is that the pending write is unblocked — and with
	// the voice held open, the read CANNOT finish unless the cancel unblocks it,
	// which is what makes this falsifiable: remove the cancel and it hangs.
	cancel()
	done := make(chan struct{})
	go func() { _, _ = io.ReadAll(r); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("cancelling did not unblock the pending write: the reader was still going after 5s")
	}
}

func TestSourceEndsAfterOneCycleUnlessRepeating(t *testing.T) {
	// UAT 83: Repeat off = one broadcast, then EOF; onSeg carries the
	// spoken length so the marquee can pace itself.
	v := &toneVoice{}
	var spoken []time.Duration
	src, _ := NewSource(v, func(context.Context) ([]Segment, error) {
		return []Segment{{Key: "a", Text: "one"}}, nil
	}, func(_ Segment, d time.Duration) { spoken = append(spoken, d) })
	src.gap = time.Millisecond
	out, err := io.ReadAll(src.Open(context.Background()))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 8000 || len(out) > 9200 { // 100 ms stereo (8820 B) + a 1 ms gap
		t.Fatalf("one cycle then EOF, got %d bytes", len(out))
	}
	if len(spoken) != 1 || spoken[0] < 90*time.Millisecond || spoken[0] > 110*time.Millisecond {
		t.Fatalf("spoken length must ride along: %v", spoken)
	}
}

func TestStateCodeAtTheStartDoesNotPanic(t *testing.T) {
	// Round 2 N-1 (found by FuzzNormalize): a block starting with a bare
	// state code sliced runes[-1:-1]. A code with nothing before it is not
	// a place; the rest of the rules still apply.
	for _, in := range []string{"CA", "CA ", "NY\nmore", "TX. Sunny.", "IN EFFECT"} {
		_ = Normalize(in)
		_ = ExpandStates(in)
	}
	if got := ExpandStates("CA"); got != "CA" {
		t.Fatalf("a bare leading code stays as written, got %q", got)
	}
}

func TestBulletLabelsAreStatements(t *testing.T) {
	// UAT 95: CAP bullets "* WHAT...Hot" read "What: Hot." — a colon, so the
	// voice states the label instead of asking "What?". Multi-word labels
	// follow; the rest of the line keeps its own casing.
	paras := Normalize("* WHAT...Hot temperatures.\n\n* WHEN...Until 8 PM PDT.\n\n* ADDITIONAL DETAILS...Drink water.")
	want := []string{"What: Hot temperatures.", "When: Until 8 PM Pacific Daylight Time.", "Additional details: Drink water."}
	if strings.Join(paras, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q want %q", paras, want)
	}
}

func TestPronounceSpellsWebAddresses(t *testing.T) {
	// UAT 95: a web address is said letter-by-letter where it must be —
	// "w w w dot weather dot gov slash sandiego" — the scheme dropped, dots,
	// slashes and dashes named; the screen keeps the address as written.
	for in, want := range map[string]string{
		"Visit www.weather.gov/sandiego for more.":         "Visit w w w dot weather dot gov slash sandiego for more.",
		"See https://forecast.weather.gov/x-y, or call.":   "See forecast dot weather dot gov slash x dash y, or call.",
		"Rain of 3.5 inches; the U.S. average is 2.1 e.g.": "Rain of 3.5 inches; the U.S. average is 2.1 e.g.", // numbers and initialisms are not addresses
		"weather.gov":                                           "weather dot gov",
		"broadcast live from KEC62, San Diego.":                 "broadcast live from K E C six two, San Diego.", // UAT 112: callsigns are spelled
		"WXJ98 and KZZ100 serve the coast.":                     "W X J nine eight and K Z Z one zero zero serve the coast.",
		"broadcasting on 165.2024 MHz, daily.":                  "broadcasting on one six five dot two zero two four mega hertz, daily.",      // UAT 112.2
		"Rain of 3.5 inches fell.":                              "Rain of 3.5 inches fell.",                                                   // only before MHz
		"via NOAA radio devices; NOAA.":                         "via Noah radio devices; Noah.",                                              // UAT 113: NOAA is a word
		"closed on SWY S-2 near Palomar Mountain Rd. and I-15;": "closed on State Highway S-2 near Palomar Mountain Road. and Interstate 15;", // UAT 120: road abbreviations
		"US-101 at Hwy 76, Blvd and St.":                        "U S 101 at Highway 76, Boulevard and Street.",
	} {
		if got := Pronounce(in); got != want {
			t.Fatalf("Pronounce(%q)\n got %q\nwant %q", in, got, want)
		}
	}
}

func TestPronounceIsVoiceOnly(t *testing.T) {
	// UAT 83: "CLI" is said "C L I"; the display text is untouched.
	if got := Pronounce("your Watchpost CLI application settings."); got != "your Watchpost C L I application settings." {
		t.Fatalf("got %q", got)
	}
	v := &spyVoice{}
	src, _ := NewSource(v, func(context.Context) ([]Segment, error) { return []Segment{{Key: "t", Text: "Watchpost CLI."}}, nil }, nil)
	_, _ = io.ReadAll(src.Open(context.Background()))
	if v.heard != "Watchpost C L I." {
		t.Fatalf("voice heard %q", v.heard)
	}
}

type spyVoice struct{ heard string }

func (v *spyVoice) Name() string { return "spy" }
func (v *spyVoice) Rate() int    { return 22050 }
func (v *spyVoice) Say(_ context.Context, text string) ([]byte, error) {
	v.heard = text
	return make([]byte, 400), nil
}

// markVoice renders msPerChar of a constant sample per character and
// records what it was asked to say (test double for voice hand-over).
type markVoice struct {
	name      string
	mark      int16
	msPerChar int
	rate      int // 0 = 22050
	mu        sync.Mutex
	said      []string
}

func (v *markVoice) Name() string { return v.name }
func (v *markVoice) Rate() int {
	if v.rate > 0 {
		return v.rate
	}
	return 22050
}
func (v *markVoice) Say(_ context.Context, text string) ([]byte, error) {
	v.mu.Lock()
	v.said = append(v.said, text)
	v.mu.Unlock()
	n := 22050 * v.msPerChar * len(text) / 1000
	out := make([]byte, n*2)
	for i := range n {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(v.mark))
	}
	return out, nil
}

func (v *markVoice) texts() []string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return append([]string(nil), v.said...)
}

func TestRemainderResumesAtAWordBoundary(t *testing.T) {
	// UAT 94: the spot is the time fraction mapped to words; too little
	// left is nothing (the next segment follows at once).
	text := "one two three four five six seven eight nine ten"
	if got := Remainder(text, 0.42); got != "five six seven eight nine ten" {
		t.Fatalf("42 %% in: %q", got)
	}
	if got := Remainder(text, 0); got != text {
		t.Fatal("the start is the whole line")
	}
	if got := Remainder(text, 0.85); got != "" {
		t.Fatalf("under three words left is nothing, got %q", got)
	}
	if got := Remainder("", 0.5); got != "" {
		t.Fatal("empty stays empty")
	}
	// The hand-over line moved to Composer.HandoffLine, which reads the script
	// tree and falls back to the built-in wording.
	if got := std.HandoffLine("Alpha", "Bravo"); got != "This is Bravo, taking over for Alpha." {
		t.Fatalf("hand-over line: %q", got)
	}
}

func TestRecastHandsOverMidSegmentAtTheSameSpot(t *testing.T) {
	// UAT 94, now through the cast: the listener saves a new assignment while a
	// segment plays. That is a HARD change — they are waiting to hear it — so
	// the running segment hands over at the spot reached, and the introduction
	// and the remainder arrive as ONE utterance so there is no gap between
	// "This is Bravo" and the words.
	a := &markVoice{name: "Alpha", mark: 1000, msPerChar: 10}
	b := &markVoice{name: "Bravo", mark: 2000, msPerChar: 10}
	long := "the quick brown fox jumps over the lazy dog while the weather radio keeps on reading the zone forecast for the coast" // 116 chars ≈ 1.16 s in Alpha
	var spoken []string
	var mu sync.Mutex
	src, err := NewSource(a, func(context.Context) ([]Segment, error) {
		return []Segment{{Key: "a", Text: long}, {Key: "b", Text: "second segment here"}, {Key: "tail", Text: "This is " + VoiceToken + " signing off."}}, nil
	}, func(seg Segment, _ time.Duration) { mu.Lock(); spoken = append(spoken, seg.Text); mu.Unlock() })
	if err != nil {
		t.Fatal(err)
	}
	src.gap = time.Millisecond

	// The resolver is the cast: it answers with whoever is assigned NOW.
	var rmu sync.Mutex
	current := Voice(a)
	src.SetResolver(func(cast.Role) (Voice, error) { rmu.Lock(); defer rmu.Unlock(); return current, nil })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := src.Open(ctx)
	head := make([]byte, 22050*3/10*4) // 300 ms in Alpha
	if _, err := io.ReadFull(r, head); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint16(head[len(head)-4:]) != 1000 {
		t.Fatal("Alpha plays first")
	}

	rmu.Lock()
	current = b
	rmu.Unlock()
	src.Recast() // the listener pressed Save

	// Within two 100 ms chunks (one may already be in flight) Bravo is heard.
	switchedAt := -1
	for i := 0; i < 3 && switchedAt < 0; i++ { // bounded probe
		chunk := make([]byte, 22050/10*4)
		if _, err := io.ReadFull(r, chunk); err != nil {
			t.Fatal(err)
		}
		for j := 0; j+4 <= len(chunk); j += 4 {
			if binary.LittleEndian.Uint16(chunk[j:]) == 2000 {
				switchedAt = i
				break
			}
		}
	}
	if switchedAt < 0 {
		t.Fatal("Bravo must be heard within two chunks of the recast")
	}
	rest, err := io.ReadAll(r) // to the end of the cycle (Repeat off)
	if err != nil {
		t.Fatal(err)
	}
	for j := 0; j+4 <= len(rest); j += 4 {
		if v := binary.LittleEndian.Uint16(rest[j:]); v != 2000 && v != 0 {
			t.Fatalf("after the recast only Bravo (and gaps) plays, saw %d at byte %d of %d; bravo said %q", v, j, len(rest), b.texts())
		}
	}

	// Bravo renders three things: the take-over utterance (introduction +
	// remainder, together), the next segment, and the tail in its own name.
	// Membership, not order — render-ahead and the writer interleave.
	// WHAT WAS HEARD, NOT WHAT WAS RENDERED — F-14, and the whole of its
	// three-sighting flake.
	//
	// This used to classify `b.texts()`, which is what Bravo was asked to
	// RENDER. The render goroutine works a segment ahead and PRE-RENDERS a
	// hand-over line in case the next segment needs one; when the voice has
	// already changed mid-segment it is not needed, `announce` skips it, and
	// nothing is played — but the render happened and `b.texts()` records it.
	// So Bravo appeared to introduce itself twice, and the old classifier
	// (`default: takeover = x`) kept whichever came LAST. Which one that was
	// depended on scheduling, so the test passed or failed by timing while the
	// product was correct both ways.
	//
	// The marquee record is what the listener actually got: onSeg fires only for
	// what is written. Classifying that pins the rule — ONE introduction is
	// heard — instead of an artefact of look-ahead.
	mu.Lock()
	played := append([]string(nil), spoken...)
	mu.Unlock()
	const intro = "This is Bravo, taking over for Alpha."
	var takeovers []string
	for _, x := range played {
		if strings.HasPrefix(x, intro) {
			takeovers = append(takeovers, x)
		}
	}
	if len(takeovers) != 1 {
		t.Fatalf("the listener heard %d introductions, want exactly one — nobody introduces themselves "+
			"twice (source.go takeOver). Everything that reached the marquee: %q", len(takeovers), played)
	}
	takeover := takeovers[0]
	if !strings.HasPrefix(takeover, intro+" ") {
		t.Fatalf("the take-over is ONE utterance opening with the introduction, got %q.\n"+
			"A BARE introduction means the recast was seen at a segment BOUNDARY rather than inside one, "+
			"so the fixture did not exercise the rule (F-14). Heard: %q", takeover, played)
	}
	remainder := strings.TrimPrefix(takeover, intro+" ")
	at := strings.Index(long, remainder)
	if at <= 0 || !strings.HasSuffix(long, remainder) || long[at-1] != ' ' || at < len(long)*2/10 || at > len(long)*6/10 {
		t.Fatalf("the remainder starts at the word reached (20–60 %% in, on a word boundary), got %q", remainder)
	}
	if got := a.texts(); len(got) < 1 || got[0] != long {
		t.Fatalf("Alpha rendered the first segment: %q", got)
	}
	if len(played) != 4 || played[0] != long || played[1] != takeover ||
		played[2] != "second segment here" || played[3] != "This is Bravo signing off." {
		t.Fatalf("the marquee follows: line, take-over, next, substituted tail — got %q", played)
	}
}

// brokenVoice cannot render (an uninstalled `say` voice, a broken Piper).
type brokenVoice struct{}

func (brokenVoice) Name() string { return "Broken" }
func (brokenVoice) Rate() int    { return 22050 }
func (brokenVoice) Say(context.Context, string) ([]byte, error) {
	return nil, errors.New("say: voice not installed")
}

func TestSourceReportsARenderFailureInsteadOfCompleting(t *testing.T) {
	// Red-team 0.9.0 C-4/F3: a voice that cannot render ends the stream, but
	// Err() says why — the deck must never read it as "broadcast complete"
	// (which under Repeat: Watchlist would spin through every favourite).
	src, _ := NewSource(brokenVoice{}, func(context.Context) ([]Segment, error) { return []Segment{{Key: "a", Text: "x"}}, nil }, nil)
	out, _ := io.ReadAll(src.Open(context.Background()))
	if len(out) != 0 {
		t.Fatal("nothing plays")
	}
	if err := src.Err(); err == nil || !strings.Contains(err.Error(), "voice not installed") {
		t.Fatalf("Err carries the render failure, got %v", err)
	}
	ok, _ := NewSource(&toneVoice{}, func(context.Context) ([]Segment, error) { return []Segment{{Key: "a", Text: "x"}}, nil }, nil)
	_, _ = io.ReadAll(ok.Open(context.Background()))
	if ok.Err() != nil {
		t.Fatal("a natural end has no error")
	}
	// TWO DIFFERENT FAILURES, TWO DIFFERENT RULES (the batch's contract 2).
	// Above: a SEGMENT that cannot be rendered ends the broadcast, because
	// there is nothing to play. Here: a listener recasts to a broken voice
	// mid-segment. The take-over cannot render — but the broadcast is NOT
	// ended, because FR-5's "never silence" is better served by carrying on.
	// A test that asserted the segment rule here would be asserting the wrong
	// contract of the wrong failure.
	long := &markVoice{name: "Alpha", mark: 1000, msPerChar: 10}
	src2, _ := NewSource(long, func(context.Context) ([]Segment, error) {
		return []Segment{{Key: "a", Text: strings.Repeat("word ", 60)}}, nil
	}, nil)
	var rmu sync.Mutex
	current := Voice(long)
	src2.SetResolver(func(cast.Role) (Voice, error) { rmu.Lock(); defer rmu.Unlock(); return current, nil })
	r := src2.Open(context.Background())
	head := make([]byte, 22050*3/10*4)
	if _, err := io.ReadFull(r, head); err != nil {
		t.Fatal(err)
	}
	rmu.Lock()
	current = brokenVoice{}
	rmu.Unlock()
	src2.Recast()
	_, _ = io.ReadAll(r)
	if err := src2.Err(); err != nil {
		t.Fatalf("a failed TAKE-OVER must not end the broadcast (contract 2), got %v", err)
	}
}

// The rate-refusal pin retired with SetVoice. A cast whose voices differ in
// sample rate is out of scope for 0.14.0 — the stream's rate is fixed when it
// opens — and a resampling decorator is on the backlog. Nothing asserts a
// refusal because nothing refuses: there is no longer an API to hand a
// different-rate voice to a running stream.

func TestSampleLine(t *testing.T) {
	if std.Sample("Alex") != "This is Alex for Watchpost Weather Radio." {
		t.Fatal(std.Sample("Alex"))
	}
	v := &spyVoice{}
	pcm, err := std.SamplePCM(context.Background(), v)
	if err != nil || len(pcm) != 800 || v.heard != "This is spy for Watchpost Weather Radio." {
		t.Fatalf("sample: %d bytes, heard %q, err %v", len(pcm), v.heard, err)
	}
}

func TestLeadPausesBeforeTheForecastSpan(t *testing.T) {
	// UAT 112.3: two seconds of air between "life safety use." and "This
	// forecast is from…" — the notice segment carries the pause, the span
	// segment none.
	segs := std.Compose(snapshot.Location{Label: "Oceanside, CA"}, nil, time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC), true, "Samantha", Station{}, Reports{}, render.Clock12)
	if len(segs) < 2 || segs[0].Pause != 2*time.Second || segs[1].Pause != time.Second /* nothing follows but the tail: the 1 s tail pause (UAT 115) */ || !strings.HasSuffix(segs[0].Text, "life safety use.") || !strings.HasPrefix(segs[1].Text, "This forecast is from") {
		t.Fatalf("lead segments: %+v", segs[:2])
	}
}

func TestReportsAreSeparatedByAir(t *testing.T) {
	// UAT 115: 2 s between the forecast and the fire report, 1 s before the
	// tail; with no fire report the forecast itself gets the 1 s.
	now := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	loc := snapshot.Location{Label: "Oceanside, CA"}
	products := []Product{{ID: "p1", Type: "ZFP", Text: ".TONIGHT...Mostly clear.\n\n$$"}}
	fire := FireReport{Known: true, RadiusKm: 25, Sources: []string{"NOAA's Hazard Mapping System"}, State: snapshot.FireState{AsOf: now}}
	segs := std.Compose(loc, products, now, true, "Samantha", Station{}, Reports{Fire: fire}, render.Clock12)
	byKey := func(prefix string) []Segment {
		var out []Segment
		for _, s := range segs {
			if strings.HasPrefix(s.Key, prefix) {
				out = append(out, s)
			}
		}
		return out
	}
	zfp, fireSegs := byKey("ZFP:"), byKey("fire:")
	if zfp[len(zfp)-1].Pause != 2*time.Second {
		t.Fatalf("the forecast's last segment pauses 2 s before the fire report: %+v", zfp[len(zfp)-1])
	}
	if fireSegs[len(fireSegs)-1].Pause != time.Second {
		t.Fatalf("the last report pauses 1 s before the tail: %+v", fireSegs[len(fireSegs)-1])
	}
	segs = std.Compose(loc, products, now, true, "Samantha", Station{}, Reports{}, render.Clock12)
	zfp = byKey("ZFP:")
	// "tail:{{voice}}", not "tail:Samantha": from 0.14.0 the voice lives in the
	// CACHE key, so a hand-over cannot mint a new sign-off segment.
	if zfp[len(zfp)-1].Pause != time.Second || segs[len(segs)-1].Key != "tail:"+VoiceToken {
		t.Fatalf("no fire report: the forecast pauses 1 s then the tail: %+v", zfp[len(zfp)-1])
	}
}
