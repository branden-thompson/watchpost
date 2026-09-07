package synth

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
)

// castVoice records what it said and, crucially, WHERE it was asked from.
type castVoice struct {
	name string
	rate int

	mu          sync.Mutex
	said        []string
	writerCalls int // Says made on the writer goroutine — R6's forbidden number
}

func newCastVoice(name string) *castVoice { return &castVoice{name: name, rate: 22050} }

func (v *castVoice) Name() string { return v.name }
func (v *castVoice) Rate() int {
	if v.rate == 0 {
		return 22050
	}
	return v.rate
}

func (v *castVoice) Say(ctx context.Context, text string) ([]byte, error) {
	v.mu.Lock()
	v.said = append(v.said, text)
	if OnWriter(ctx) {
		v.writerCalls++
	}
	v.mu.Unlock()
	return make([]byte, 2*len(text)*100), nil // ~100 mono samples per character
}

func (v *castVoice) texts() []string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return append([]string(nil), v.said...)
}

func (v *castVoice) onWriter() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.writerCalls
}

// byRole is a resolver over a fixed assignment, with a fallback.
func byRole(fallback Voice, m map[cast.Role]Voice) func(cast.Role) (Voice, error) {
	return func(r cast.Role) (Voice, error) {
		if v, ok := m[r]; ok {
			return v, nil
		}
		return fallback, nil
	}
}

// drain plays a source to its end and returns the marquee lines it emitted.
func drain(t *testing.T, src *Source) []string {
	t.Helper()
	var mu sync.Mutex
	var lines []string
	src.onSeg = func(seg Segment, _ time.Duration) { mu.Lock(); lines = append(lines, seg.Text); mu.Unlock() }
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err := io.ReadAll(src.Open(ctx)); err != nil {
		t.Fatalf("read: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	return append([]string(nil), lines...)
}

// --- Task 2.2: a voice per role ---

func TestEachSegmentRendersInItsRolesVoice(t *testing.T) {
	root, alerts, fire := newCastVoice("Root"), newCastVoice("Alerts"), newCastVoice("Fire")
	src, err := NewSource(root, func(context.Context) ([]Segment, error) {
		return []Segment{
			{Key: "wx", Text: "the forecast", Role: cast.Weather},
			{Key: "alert", Text: "a warning", Role: cast.Breaking},
			{Key: "fire", Text: "the fire report", Role: cast.Fire},
		}, nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	src.gap = time.Millisecond
	src.SetResolver(byRole(root, map[cast.Role]Voice{cast.Breaking: alerts, cast.Fire: fire}))
	drain(t, src)

	if got := root.texts(); len(got) != 1 || got[0] != "the forecast" {
		t.Errorf("the root reads the unassigned role: %q", got)
	}
	if got := alerts.texts(); len(got) == 0 || got[len(got)-1] != "a warning" {
		t.Errorf("the alerts voice reads the alert: %q", got)
	}
	if got := fire.texts(); len(got) == 0 || got[len(got)-1] != "the fire report" {
		t.Errorf("the fire voice reads the fire report: %q", got)
	}
}

// The cache key carries the VOICE, so two correspondents reading the same words
// keep their own audio. Before 0.14.0 the key was the segment key alone, which
// with a cast would have handed one voice's audio to another.
func TestTheCacheKeySeparatesTwoVoicesOnTheSameSegment(t *testing.T) {
	a, b := newCastVoice("Alpha"), newCastVoice("Bravo")
	// ATOMIC because the resolver runs on the RENDER goroutine while the test
	// writes this one. As a plain bool it was a real data race that only the
	// full `-race ./...` run had enough scheduling pressure to catch — green
	// five times out of five in isolation. An intermittent race in a test
	// reddens CI on someone else's commit.
	var flip atomic.Bool
	src, _ := NewSource(a, func(context.Context) ([]Segment, error) {
		return []Segment{{Key: "same", Text: "identical words", Role: cast.Weather}}, nil
	}, nil)
	src.gap = time.Millisecond
	src.Loop(true)
	src.SetResolver(func(cast.Role) (Voice, error) {
		if flip.Load() {
			return b, nil
		}
		return a, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	r := src.Open(ctx)
	buf := make([]byte, 4096)
	if _, err := io.ReadFull(r, buf); err != nil {
		t.Fatal(err)
	}
	flip.Store(true)
	// Read enough for several more cycles, then stop.
	for range 40 {
		if _, err := io.ReadFull(r, buf); err != nil {
			break
		}
	}
	cancel()

	if len(a.texts()) == 0 {
		t.Fatal("Alpha never rendered")
	}
	if len(b.texts()) == 0 {
		t.Fatal("Bravo never rendered the same segment — the cache handed it Alpha's audio")
	}
	entries, bytes := src.Cached()
	if entries < 2 {
		t.Errorf("the cache holds %d entries; the same segment in two voices is two entries", entries)
	}
	if bytes <= 0 {
		t.Error("Cached must report live bytes")
	}
}

func TestCachedIsMaintainedNotRecomputed(t *testing.T) {
	v := newCastVoice("One")
	src, _ := NewSource(v, func(context.Context) ([]Segment, error) { return nil, nil }, nil)
	ctx := context.Background()
	for i, text := range []string{"first", "second", "third"} {
		if _, err := src.renderText(ctx, "k"+text, v, text); err != nil {
			t.Fatal(err)
		}
		entries, bytes := src.Cached()
		if entries != i+1 {
			t.Errorf("after %d renders the cache holds %d entries", i+1, entries)
		}
		var want int
		src.mu.Lock()
		for _, pcm := range src.cache {
			want += len(pcm)
		}
		src.mu.Unlock()
		if bytes != want {
			t.Errorf("Cached reports %d bytes, the map holds %d — the running count has drifted", bytes, want)
		}
	}
	// Re-rendering a key already present must not double-count it.
	if _, err := src.renderText(ctx, "kfirst", v, "first"); err != nil {
		t.Fatal(err)
	}
	if entries, _ := src.Cached(); entries != 3 {
		t.Errorf("a cache hit added an entry: %d", entries)
	}
}

func TestTheCacheIsBoundedInBytesAndEvictsOldestFirst(t *testing.T) {
	v := newCastVoice("Big")
	src, _ := NewSource(v, func(context.Context) ([]Segment, error) { return nil, nil }, nil)
	ctx := context.Background()
	// Each render is ~200 KB (100 mono samples a character, 2 bytes each).
	big := strings.Repeat("x", 1000)
	for i := range 260 { // ~52 MB, past the 40 MB bound
		if _, err := src.renderText(ctx, "k"+string(rune('a'+i%26))+strings.Repeat("z", i), v, big); err != nil {
			t.Fatal(err)
		}
	}
	entries, bytes := src.Cached()
	if bytes > maxCachedBytes {
		t.Errorf("the cache holds %d bytes, the bound is %d", bytes, maxCachedBytes)
	}
	if entries == 0 {
		t.Error("eviction must not empty the cache")
	}
	// The oldest key went first.
	src.mu.Lock()
	_, oldestStillThere := src.cache["ka"]
	src.mu.Unlock()
	if oldestStillThere {
		t.Error("eviction is oldest-first; the first key survived")
	}
}

// spokenName is the one owner of a name that will be spoken or drawn (RS-18).
func TestSpokenNameIsPlainBoundedAndNamesTheUnnamed(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Samantha", "Samantha"},
		{"", unnamedCorrespondent},
		{"  Daniel  ", "Daniel"},
		{"Two\nLines", "Two Lines"},
	} {
		if got := spokenName(&castVoice{name: tc.in}); got != tc.want {
			t.Errorf("spokenName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if got := spokenName(nil); got != unnamedCorrespondent {
		t.Errorf("spokenName(nil) = %q", got)
	}
	long := spokenName(&castVoice{name: strings.Repeat("A", 500)})
	if len([]rune(long)) != maxSpokenName {
		t.Errorf("a long name is capped at %d runes, got %d", maxSpokenName, len([]rune(long)))
	}
	// A hostile name reaches neither a synthesiser's stdin nor a frame intact.
	hostile := spokenName(&castVoice{name: "Ann\x1b[31m\x07ie"})
	if strings.ContainsAny(hostile, "\x1b\x07\n\r\t") {
		t.Errorf("spokenName let a control character through: %q", hostile)
	}
}

// --- Task 2.3: the hand-over, rendered ahead ---

func TestOneHandOverPerVoiceChangeSpokenByTheIncomingVoice(t *testing.T) {
	root, fire := newCastVoice("Samantha"), newCastVoice("Rishi")
	src, _ := NewSource(root, func(context.Context) ([]Segment, error) {
		return []Segment{
			{Key: "wx", Text: "the forecast", Role: cast.Weather},
			{Key: "fire", Text: "the fire report", Role: cast.Fire},
			{Key: "tail", Text: "signing off", Role: cast.Station},
		}, nil
	}, nil)
	src.gap = time.Millisecond
	src.SetResolver(byRole(root, map[cast.Role]Voice{cast.Fire: fire}))
	lines := drain(t, src)

	const intoFire = "This is Rishi, taking over for Samantha."
	const backToRoot = "This is Samantha, taking over for Rishi."
	// The incoming voice speaks its own introduction, naming the outgoing one.
	if got := fire.texts(); !containsLine(got, intoFire) {
		t.Errorf("the incoming voice speaks the hand-over: %q", got)
	}
	if got := root.texts(); !containsLine(got, backToRoot) {
		t.Errorf("the return hand-over is spoken by the returning voice: %q", got)
	}
	// Exactly one per change — two changes here, two lines on the marquee.
	if n := countLine(lines, intoFire) + countLine(lines, backToRoot); n != 2 {
		t.Errorf("want exactly two hand-overs on the marquee, got %d in %q", n, lines)
	}
	// And a cycle with no change speaks none.
	quiet, _ := NewSource(root, func(context.Context) ([]Segment, error) {
		return []Segment{{Key: "a", Text: "one", Role: cast.Weather}, {Key: "b", Text: "two", Role: cast.Fire}}, nil
	}, nil)
	quiet.gap = time.Millisecond
	quiet.SetResolver(byRole(root, nil))
	for _, l := range drain(t, quiet) {
		if strings.Contains(l, "taking over") {
			t.Errorf("one voice reading everything hands over to nobody: %q", l)
		}
	}
}

// R6, stated as the property rather than the symptom: the goroutine feeding the
// speaker makes ZERO synthesiser calls across a cycle full of hand-overs. The
// gap between writes is what a listener notices; the call count is why.
func TestWriterNeverStarvesAcrossHandOvers(t *testing.T) {
	root, fire, seismic := newCastVoice("Samantha"), newCastVoice("Rishi"), newCastVoice("Daniel")
	src, _ := NewSource(root, func(context.Context) ([]Segment, error) {
		return []Segment{
			{Key: "wx", Text: "the forecast for today", Role: cast.Weather},
			{Key: "fire", Text: "the fire report", Role: cast.Fire},
			{Key: "seismic", Text: "the seismic report", Role: cast.Seismic},
			{Key: "wx2", Text: "the forecast again", Role: cast.Weather},
			{Key: "tail", Text: "signing off", Role: cast.Station},
		}, nil
	}, nil)
	src.gap = time.Millisecond
	src.SetResolver(byRole(root, map[cast.Role]Voice{cast.Fire: fire, cast.Seismic: seismic}))
	drain(t, src)

	for _, v := range []*castVoice{root, fire, seismic} {
		if n := v.onWriter(); n != 0 {
			t.Errorf("%s was asked to render %d time(s) ON THE WRITER GOROUTINE; R6 allows zero — "+
				"every hand-over must be rendered ahead", v.Name(), n)
		}
	}
}

// A paced reader at real time sees no gap beyond the output path's slack, even
// when every render is slow. This is the symptom half of R6; the test above is
// the cause half, and both are kept because a fast machine can pass this one
// while the bug is present.
func TestAPacedReaderSeesNoGapAcrossAHandOver(t *testing.T) {
	slow := func(name string) *slowVoice {
		return &slowVoice{castVoice: castVoice{name: name, rate: 22050}, delay: 300 * time.Millisecond}
	}
	root, fire := slow("Samantha"), slow("Rishi")
	src, _ := NewSource(root, func(context.Context) ([]Segment, error) {
		return []Segment{
			{Key: "wx", Text: strings.Repeat("forecast ", 20), Role: cast.Weather},
			{Key: "fire", Text: strings.Repeat("fire ", 20), Role: cast.Fire},
		}, nil
	}, nil)
	src.gap = time.Millisecond
	src.SetResolver(byRole(root, map[cast.Role]Voice{cast.Fire: fire}))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r := src.Open(ctx)
	buf := make([]byte, 4410*4) // 200 ms of stereo at 22050
	var worst time.Duration
	last := time.Now()
	for {
		n, err := io.ReadFull(r, buf)
		if n > 0 {
			if d := time.Since(last); d > worst {
				worst = d
			}
			last = time.Now()
		}
		if err != nil {
			break
		}
	}
	// The output path holds ≈ 0.7 s; anything longer is audible silence.
	if worst > 700*time.Millisecond {
		t.Errorf("the writer stalled for %v across a hand-over; the output path holds ~700ms", worst)
	}
}

type slowVoice struct {
	castVoice
	delay time.Duration
}

func (v *slowVoice) Say(ctx context.Context, text string) ([]byte, error) {
	select {
	case <-time.After(v.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return v.castVoice.Say(ctx, text)
}

// A hand-over line that will not render leaves the marquee saying so and the
// SEGMENT still plays: contract 2's non-fatal half.
func TestAFailedHandOverLineIsNotFatal(t *testing.T) {
	root := newCastVoice("Samantha")
	src, _ := NewSource(root, func(context.Context) ([]Segment, error) {
		return []Segment{
			{Key: "wx", Text: "the forecast", Role: cast.Weather},
			{Key: "fire", Text: "the fire report", Role: cast.Fire},
		}, nil
	}, nil)
	src.gap = time.Millisecond
	// The incoming voice can read its segment but not its introduction.
	picky := &pickyVoice{castVoice: castVoice{name: "Rishi", rate: 22050}, refuse: "taking over"}
	src.SetResolver(byRole(root, map[cast.Role]Voice{cast.Fire: picky}))
	lines := drain(t, src)

	if err := src.Err(); err != nil {
		t.Fatalf("a failed hand-over LINE must not end the broadcast, got %v", err)
	}
	if !containsLine(lines, "the fire report") {
		t.Errorf("the segment must still play: %q", lines)
	}
}

// pickyVoice refuses any text containing refuse.
type pickyVoice struct {
	castVoice
	refuse string
}

func (v *pickyVoice) Say(ctx context.Context, text string) ([]byte, error) {
	if strings.Contains(text, v.refuse) {
		return nil, errors.New("say: cannot render that")
	}
	return v.castVoice.Say(ctx, text)
}

// A SEGMENT that will not render still ends the broadcast, with its reason:
// contract 2's fatal half. Two different failures, two different rules.
func TestAFailedSegmentStillEndsTheStream(t *testing.T) {
	root := newCastVoice("Samantha")
	src, _ := NewSource(root, func(context.Context) ([]Segment, error) {
		return []Segment{{Key: "a", Text: "fine", Role: cast.Weather}, {Key: "b", Text: "boom", Role: cast.Fire}}, nil
	}, nil)
	src.gap = time.Millisecond
	src.SetResolver(byRole(root, map[cast.Role]Voice{cast.Fire: &pickyVoice{castVoice: castVoice{name: "Rishi", rate: 22050}, refuse: "boom"}}))
	drain(t, src)
	if err := src.Err(); err == nil || !strings.Contains(err.Error(), "cannot render") {
		t.Fatalf("a failed SEGMENT ends the stream with its reason, got %v", err)
	}
}

// Invalidate is the soft change: it must not produce a mid-segment hand-over,
// and — with one segment of look-ahead — it does not reach the air until the
// next segment RENDERED. The assertion is about what does NOT happen: nothing
// renders on the writer, and no introduction is spoken mid-segment.
func TestInvalidateNeverRendersOnTheWriter(t *testing.T) {
	root, other := newCastVoice("Samantha"), newCastVoice("Rishi")
	var swapped bool
	var mu sync.Mutex
	src, _ := NewSource(root, func(context.Context) ([]Segment, error) {
		return []Segment{
			{Key: "a", Text: strings.Repeat("word ", 40), Role: cast.Weather},
			{Key: "b", Text: strings.Repeat("more ", 40), Role: cast.Weather},
			{Key: "c", Text: strings.Repeat("last ", 40), Role: cast.Weather},
		}, nil
	}, nil)
	src.gap = time.Millisecond
	src.SetResolver(func(cast.Role) (Voice, error) {
		mu.Lock()
		defer mu.Unlock()
		if swapped {
			return other, nil
		}
		return root, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	r := src.Open(ctx)
	head := make([]byte, 8192)
	if _, err := io.ReadFull(r, head); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	swapped = true
	mu.Unlock()
	src.Invalidate() // a host fact landed — nobody is waiting for it
	if _, err := io.ReadAll(r); err != nil {
		t.Fatal(err)
	}

	for _, v := range []*castVoice{root, other} {
		if n := v.onWriter(); n != 0 {
			t.Errorf("%s rendered %d time(s) on the writer after an Invalidate; a soft change never renders there", v.Name(), n)
		}
	}
	// The segment already rendered played as it was: the old voice is still
	// heard after the Invalidate, which is the contract, not a bug.
	if len(root.texts()) == 0 {
		t.Error("the segment rendered before the Invalidate must still have been rendered by the old voice")
	}
}

func containsLine(lines []string, want string) bool { return countLine(lines, want) > 0 }

func countLine(lines []string, want string) int {
	n := 0
	for _, l := range lines {
		if l == want {
			n++
		}
	}
	return n
}

// UAT 2026-08-30 (bug #5): "Eddie took over for Karen" and then, three seconds
// later, "This is Eddie signing off" — he introduced himself twice.
//
// It happened where a location had no seismic report, so the fire report ran
// straight into the sign-off across a voice change. The sign-off NAMES ITS OWN
// SPEAKER, so preceding it with an introduction is redundant by construction,
// not merely repetitive: the listener still hears the change of voice and is
// still told whose it is.
func TestNoHandOverIntoASegmentThatNamesItsOwnSpeaker(t *testing.T) {
	karen, eddie := newCastVoice("Karen"), newCastVoice("Eddie")
	src, _ := NewSource(karen, func(context.Context) ([]Segment, error) {
		return []Segment{
			{Key: "fire", Text: "the fire report", Role: cast.Fire},
			// No seismic report: the sign-off follows directly, in another
			// voice, and introduces itself.
			{Key: "tail:" + VoiceToken, Text: "This is " + VoiceToken + " for Watchpost Weather Radio.", Role: cast.Station, SelfIntro: true},
		}, nil
	}, nil)
	src.gap = time.Millisecond
	src.SetResolver(byRole(karen, map[cast.Role]Voice{cast.Fire: karen, cast.Station: eddie}))
	lines := drain(t, src)

	for _, l := range lines {
		if strings.Contains(l, "taking over") {
			t.Errorf("no introduction before a segment that introduces itself: %q", l)
		}
	}
	// The listener is still told who is reading — by the segment itself.
	if !containsLine(lines, "This is Eddie for Watchpost Weather Radio.") {
		t.Errorf("the sign-off must still name its speaker: %q", lines)
	}
	// And an ordinary boundary still hands over.
	ordinary, _ := NewSource(karen, func(context.Context) ([]Segment, error) {
		return []Segment{
			{Key: "fire", Text: "the fire report", Role: cast.Fire},
			{Key: "seismic", Text: "the seismic report", Role: cast.Seismic},
		}, nil
	}, nil)
	ordinary.gap = time.Millisecond
	ordinary.SetResolver(byRole(karen, map[cast.Role]Voice{cast.Fire: karen, cast.Seismic: eddie}))
	if got := drain(t, ordinary); !containsLine(got, "This is Eddie, taking over for Karen.") {
		t.Errorf("an ordinary voice change still hands over: %q", got)
	}
}
