package synth

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/plaintext"
)

// VoiceToken stands for the CURRENT voice's name, in two places.
//
//   - In a segment's TEXT it is spoken and shown, so a sign-off planned in one
//     voice is read correctly by the voice that reaches it (UAT 94).
//   - In a segment's KEY it marks where the rendering voice belongs, and the
//     Source substitutes it when composing the CACHE key.
//
// The second use is 0.14.0's, and it fixes a latent bug. The tail used to be
// keyed "tail:" + voiceName, which meant a change of correspondent minted a new
// SEGMENT identity — the segment was "different" because somebody else was
// reading it. With a cast that is wrong in a way a listener would notice: every
// hand-over would invalidate the sign-off. A segment key names WHAT is said; a
// cache key names what was said AND BY WHOM.
const VoiceToken = "{{voice}}"

// writerKey marks the context the WRITER goroutine renders under.
//
// It exists so a test can assert the property R6 actually needs — *zero
// synthesiser calls on the goroutine feeding the speaker* — rather than
// inferring it from a stopwatch. The gap between writes is the symptom; the
// call count is the cause, and a symptom-only test passes on a fast machine
// while the bug is still there.
type writerKey struct{}

func withWriter(ctx context.Context) context.Context {
	return context.WithValue(ctx, writerKey{}, true)
}

// OnWriter reports whether ctx belongs to the writer goroutine. Test voices
// read it to count what R6 forbids.
func OnWriter(ctx context.Context) bool {
	on, _ := ctx.Value(writerKey{}).(bool)
	return on
}

// Source is the synthesized broadcast as a PCM stream (16-bit LE STEREO at the
// stream's rate): it narrates one cycle, then — when Repeat is on — asks for
// the next cycle's segments and continues; a newly issued product never swaps
// mid-segment (§10.4).
//
// From 0.14.0 each segment carries a ROLE and the Source asks a resolver who
// reads it. Two correspondents in one cycle hand over to each other by name,
// and — this is the part R6 depends on — THE HAND-OVER LINE IS RENDERED AHEAD,
// on the render goroutine, at the boundary where the voice changes. The writer
// only writes it. A hand-over rendered on the writer would cost 2–20 s of
// silence per boundary on Linux, where a Piper render is a model load.
type Source struct {
	next     func(ctx context.Context) ([]Segment, error) // the next cycle's segments
	onSeg    func(Segment, time.Duration)                 // narration text + its spoken length, for the marquee
	gap      time.Duration
	rate     int   // fixed at construction: the stream's rate cannot change mid-broadcast
	fallback Voice // reads every role until a resolver is installed

	mu      sync.Mutex
	resolve func(cast.Role) (Voice, error)
	handoff func(from, to string) string // the scripted hand-over line; nil = the built-in

	// Two generations, because two different things change a voice and they
	// must behave differently (the batch's contract 1).
	//
	//   softGen — a HOST FACT landed: discovery finished, an install completed.
	//     Nobody asked for it and nobody is waiting for it, so it takes effect
	//     at the next segment RENDERED. The segment already rendered plays as
	//     it is, and THE WRITER NEVER RE-RENDERS FOR IT.
	//   hardGen — the LISTENER saved a cast. They are waiting to hear it, so
	//     the running segment hands over at the spot reached. This is the one
	//     render the writer is allowed to perform, and it is non-fatal.
	softGen uint64
	hardGen uint64

	// onAir is the voice the LISTENER last actually heard. A hand-over is
	// spoken when the voice on air changes — which is FR-5 stated exactly, and
	// it is the only rule that gets both paths right. Render order and playing
	// order are the same until a recast lands, and then they are not: several
	// segments may already be voiced by the outgoing correspondent, and each
	// must switch, but only the FIRST of them introduces anybody.
	onAir Voice

	repeat bool
	cache  map[string][]byte // (segment key, voice name) -> rendered mono PCM
	order  []string          // insertion order, for oldest-first eviction
	bytes  int               // the cache's live size, maintained rather than recomputed
	err    error             // why the broadcast ended early; nil = it ran to its end
}

// Err is why the stream ended before its cycle did — a SEGMENT that could not
// be rendered — or nil for a natural end. The deck reads it when the engine
// reports the end (red-team 0.9.0 C-4): a render failure must never pass for
// "broadcast complete", which under Repeat: Watchlist would spin through every
// favourite.
//
// A failed HAND-OVER LINE never sets it. Those are two different failures and
// they get two different rules (contract 2): a segment's failure leaves nothing
// to play, so the broadcast ends; a hand-over line's leaves the segment
// perfectly playable, and FR-5's "never silence" is better served by continuing
// without the introduction than by stopping the broadcast over it.
func (s *Source) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// NewSource builds a source. voice reads every role until SetResolver is
// called, so a caller with no cast still gets 0.13.0's behaviour. onSeg may be
// nil.
func NewSource(voice Voice, next func(context.Context) ([]Segment, error), onSeg func(Segment, time.Duration)) (*Source, error) {
	if err := invariant.Check(voice != nil && next != nil, "synth: voice and segment provider are required"); err != nil {
		return nil, err
	}
	if onSeg == nil {
		onSeg = func(Segment, time.Duration) {}
	}
	return &Source{
		fallback: voice, rate: voice.Rate(), next: next, onSeg: onSeg,
		gap: 400 * time.Millisecond, cache: map[string][]byte{},
	}, nil
}

// SetResolver installs the cast: who reads each role, on this host. Until it is
// called every role is read by the voice NewSource was given.
func (s *Source) SetResolver(r func(cast.Role) (Voice, error)) {
	s.mu.Lock()
	s.resolve = r
	s.mu.Unlock()
}

// SetHandoffLine installs the scripted hand-over line. nil restores the
// built-in, which is also what an empty return falls back to — a broken
// override must never become silence.
func (s *Source) SetHandoffLine(f func(from, to string) string) {
	s.mu.Lock()
	s.handoff = f
	s.mu.Unlock()
}

// Invalidate is the SOFT change: a host fact landed. It takes effect at the
// next segment RENDERED — with one segment of look-ahead, two segments later —
// and the segment already rendered plays exactly as it was rendered.
//
// That look-ahead delay is a discovered contract, not an accident: it was found
// by running the design, and a test asserting "the very next segment" would be
// asserting something the architecture cannot deliver without making the writer
// render, which is what R6 forbids.
func (s *Source) Invalidate() {
	s.mu.Lock()
	s.softGen++
	s.mu.Unlock()
}

// Recast is the HARD change: the listener saved a cast and is waiting to hear
// it. The running segment hands over at the spot reached.
func (s *Source) Recast() {
	s.mu.Lock()
	s.hardGen++
	s.mu.Unlock()
}

// generations reads both counters.
func (s *Source) generations() (soft, hard uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.softGen, s.hardGen
}

// voiceFor resolves the voice that reads role. A resolver that fails, or is not
// installed, yields the fallback: cast.Resolve has already walked the tree, and
// this is the last guard against a role with nobody to read it.
func (s *Source) voiceFor(role cast.Role) (Voice, error) {
	s.mu.Lock()
	resolve, fallback := s.resolve, s.fallback
	s.mu.Unlock()
	if resolve == nil {
		return fallback, nil
	}
	v, err := resolve(role)
	if err != nil || v == nil {
		if fallback == nil {
			return nil, fmt.Errorf("synth: no voice for %s: %w", role.Key(), err)
		}
		return fallback, nil
	}
	return v, nil
}

// handoffLine is the line the incoming correspondent opens with.
func (s *Source) handoffLine(from, to string) string {
	s.mu.Lock()
	f := s.handoff
	s.mu.Unlock()
	if f != nil {
		if line := f(from, to); line != "" {
			return line
		}
	}
	return fmt.Sprintf(handoverBuiltin, to, from)
}

const (
	// unnamedCorrespondent is what a voice with no name calls itself — the
	// macOS System Voice, which modern macOS will not name (UAT 88).
	unnamedCorrespondent = "your correspondent"
	// maxSpokenName bounds a name that will be spoken and drawn. Long enough
	// for "Aman (English (India))", short enough that a hostile name cannot
	// push a marquee row off the screen.
	maxSpokenName = 48
)

// spokenName is the ONE owner of a voice's name as it will be spoken or shown
// (RS-18). A voice name reaches a synthesiser's stdin and a terminal frame, and
// it can come from a hand-edited config, so it is made plain here rather than
// at each of those seams.
func spokenName(v Voice) string {
	if v == nil {
		return unnamedCorrespondent
	}
	name := plaintext.Line(v.Name())
	if name == "" {
		return unnamedCorrespondent
	}
	if r := []rune(name); len(r) > maxSpokenName {
		name = string(r[:maxSpokenName])
	}
	return name
}

// Loop makes the broadcast repeat (true) or end after the current cycle.
func (s *Source) Loop(on bool) {
	s.mu.Lock()
	s.repeat = on
	s.mu.Unlock()
}

func (s *Source) repeating() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.repeat
}

// Rate is the PCM rate, fixed for the stream's life. A cast whose voices differ
// in rate is out of scope for 0.14.0: a resampling decorator is on the backlog,
// and until it lands the stream keeps the rate it opened with.
func (s *Source) Rate() int { return s.rate }

// duration is the spoken length of stereo PCM at the stream's rate.
func (s *Source) duration(pcm []byte) time.Duration {
	return time.Duration(float64(len(pcm)/4) / float64(s.Rate()) * float64(time.Second))
}

// Open returns a reader that yields the broadcast until ctx ends (or the cycle
// ends with Repeat off). Segments are rendered one ahead of playback (UAT 81:
// no pause between segments), and cancelling ctx unblocks any pending write at
// once (fast stop).
func (s *Source) Open(ctx context.Context) io.Reader {
	pr, pw := io.Pipe()
	rendered := make(chan renderedSeg, 1) // one segment of look-ahead
	go s.renderLoop(ctx, rendered)
	go func() {
		defer func() { _ = pw.Close() }()
		writerCtx := withWriter(ctx)
		for r := range rendered {
			if !s.play(writerCtx, pw, r) {
				return
			}
		}
	}()
	go func() {
		<-ctx.Done()
		_ = pr.CloseWithError(ctx.Err()) // unblock a writer mid-segment
	}()
	return pr
}

// renderedSeg is one segment, its audio, who read it, and — when the voice
// changed at this boundary — the hand-over line ALREADY RENDERED.
type renderedSeg struct {
	seg   Segment
	pcm   []byte
	voice Voice
	soft  uint64 // the soft generation in force when it rendered
	hard  uint64 // and the hard one — a segment rendered before a recast must still hand over

	// The introduction the incoming correspondent speaks before this segment,
	// rendered on the render goroutine. Empty text = the voice did not change
	// here. handoffErr records a failure the writer reports WITHOUT ending the
	// broadcast.
	handoffText string
	handoffPCM  []byte
	handoffErr  error
}

// renderLoop plans cycles, resolves each segment's voice, renders the segment
// AND any hand-over line that precedes it, and hands both to the writer.
func (s *Source) renderLoop(ctx context.Context, out chan<- renderedSeg) {
	defer close(out)
	var lastVoice Voice
	// Bounded per P10-02: each cycle re-plans; the loop ends with ctx, or after
	// one cycle when Repeat is off.
	for cycle := 0; ctx.Err() == nil && cycle < 1<<20; cycle++ {
		segs, err := s.next(ctx)
		if err != nil || len(segs) == 0 {
			s.onSeg(Segment{Key: "waiting", Text: "Waiting for the forecast products…"}, 0) // the marquee says why it is quiet (C-11)
			if !sleepCtx(ctx, 30*time.Second) {
				return
			}
			continue
		}
		for _, seg := range segs {
			r, ok := s.renderAhead(ctx, seg, lastVoice)
			if !ok {
				return // a SEGMENT failed: fail() recorded why and the stream ends
			}
			lastVoice = r.voice
			select {
			case out <- r:
			case <-ctx.Done():
				return
			}
		}
		if !s.repeating() {
			return // one broadcast, then the stream ends (UAT 83)
		}
	}
}

// renderAhead resolves one segment's voice, renders any hand-over the change of
// voice calls for, then renders the segment itself. It reports false only when
// the SEGMENT could not be rendered.
func (s *Source) renderAhead(ctx context.Context, seg Segment, lastVoice Voice) (renderedSeg, bool) {
	// The hard generation is read BEFORE the voice is resolved, not after the
	// render. A recast landing in between would otherwise stamp a segment
	// voiced by the OUTGOING correspondent with the INCOMING generation: the
	// writer would see it as current, play it as rendered, and the listener
	// would hear the old voice with no hand-over immediately after saving.
	// (Found as an intermittent test failure; the window is a few microseconds
	// wide and would have been a rare, unreproducible field report.)
	_, hardAtStart := s.generations()
	v, err := s.voiceFor(seg.Role)
	if err != nil {
		return renderedSeg{}, s.fail(err)
	}
	r := renderedSeg{seg: seg, voice: v}
	// No hand-over into a segment that introduces its own speaker: the sign-off
	// says "This is Eddie for Watchpost Weather Radio", so preceding it with
	// "This is Eddie, taking over for Karen" introduces him twice in a row.
	// The listener still hears the change of voice AND is told whose it is —
	// the segment does both, which is why the introduction is redundant rather
	// than merely repetitive.
	if lastVoice != nil && !seg.SelfIntro && spokenName(lastVoice) != spokenName(v) {
		r.handoffText = s.handoffLine(spokenName(lastVoice), spokenName(v))
		// Rendered HERE, ahead of the air, and cached per (from → to, line) so
		// a cycle that crosses the same boundary twice renders it once.
		r.handoffPCM, r.handoffErr = s.renderText(ctx, handoffKey(lastVoice, v, r.handoffText), v, r.handoffText)
	}
	pcm, soft, err := s.render(ctx, seg, v)
	if err != nil {
		return renderedSeg{}, s.fail(err)
	}
	r.pcm, r.soft, r.hard = pcm, soft, hardAtStart
	return r, true
}

// handoffKey identifies one hand-over in the cache: who to whom, and the exact
// words, so a reworded override never reuses the old line's audio.
func handoffKey(from, to Voice, line string) string {
	return "handoff:" + spokenName(from) + "\x00" + spokenName(to) + "\x00" + line
}

// play streams one rendered segment to the pipe in 100 ms chunks, then the
// inter-segment gap.
//
// What it does NOT do is render, except in the one case the listener is waiting
// for. It writes the hand-over the render goroutine prepared; it plays a
// segment whose soft generation is stale exactly as rendered; and only a HARD
// generation change — the listener's own save — makes it synthesise, which it
// does once, folding the introduction and the remainder into a single Say.
//
// False when the pipe is gone (stop) or a SEGMENT could not be rendered.
func (s *Source) play(ctx context.Context, pw *io.PipeWriter, r renderedSeg) bool {
	// The hard generation is the one THIS SEGMENT WAS RENDERED UNDER, not the
	// one in force now. With a segment of look-ahead, a recast can land after
	// the next segment is already voiced by the outgoing correspondent; that
	// segment must hand over too, from its very first word, or the listener
	// hears the old voice again after the change they just made.
	hard := r.hard
	_, now := s.generations()
	pcm := monoToStereo(r.pcm)
	text := s.spoken(r.seg.Text, r.voice)
	// A STALE segment — one voiced before a recast the listener has since made
	// — is about to be taken over in full. It neither announces nor reaches the
	// marquee as rendered: takeOver does both, once, in the voice that will
	// actually speak. Doing either here would introduce twice and show a line
	// nobody hears.
	if hard == now {
		if !s.announce(pw, r) {
			return false
		}
		s.onSeg(Segment{Key: r.seg.Key, Text: text, Role: r.seg.Role}, s.duration(pcm))
	}

	chunk := s.Rate() / 10 * 4
	// Bounded: one chunk per pass, plus at most one take-over per hard change.
	for written := 0; written < len(pcm); {
		if _, h := s.generations(); h != hard {
			hard = h
			next := s.takeOver(ctx, r, float64(written)/float64(len(pcm)))
			if next == nil {
				break // nothing worth handing over: the next segment follows
			}
			// text is not carried forward: takeOver re-derived the remainder
			// against the INCOMING voice and has already put it on the marquee.
			pcm, written = next, 0
			continue
		}
		n := min(chunk, len(pcm)-written)
		if _, err := pw.Write(pcm[written : written+n]); err != nil {
			return false
		}
		written += n
	}
	_, err := pw.Write(s.silence(s.gap + r.seg.Pause)) // the segment's own pause rides after the standard gap (UAT 112.3)
	return err == nil
}

// changingVoice reports whether v is a different correspondent from the one the
// listener last heard, and records v as the one now on air.
func (s *Source) changingVoice(v Voice) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := s.onAir != nil && spokenName(s.onAir) != spokenName(v)
	s.onAir = v
	return changed
}

// announce writes the hand-over line the render goroutine prepared.
//
// A line that failed to render is reported on the marquee and the segment plays
// anyway (contract 2). This is the only place that rule is applied, so nothing
// can accidentally end a broadcast over a missing introduction.
func (s *Source) announce(pw *io.PipeWriter, r renderedSeg) bool {
	changing := s.changingVoice(r.voice)
	if r.handoffText == "" || !changing {
		return true // no change on air: whatever the render goroutine prepared is stale
	}
	if r.handoffErr != nil || len(r.handoffPCM) == 0 {
		s.onSeg(Segment{Key: "handoff:failed", Text: r.handoffText, Role: r.seg.Role}, 0)
		return true // NOT fatal: the segment is perfectly playable
	}
	pcm := monoToStereo(r.handoffPCM)
	s.onSeg(Segment{Key: "handoff", Text: r.handoffText, Role: r.seg.Role}, s.duration(pcm))
	_, err := pw.Write(pcm)
	return err == nil
}

// takeOver handles the listener's own Recast mid-segment: the incoming
// correspondent introduces themselves and reads the remainder as ONE utterance,
// so there is no gap between the introduction and the words.
//
// This is the single render the writer is permitted, and it is deliberate: the
// listener has just pressed Save and is waiting to hear the result, so a pause
// here is understood where a pause at an ordinary boundary is not. It is
// non-fatal — if the new voice cannot render, the rest of THIS segment is lost
// and the broadcast continues.
//
// nil means "nothing worth handing over": the next segment is read by the new
// voice regardless.
func (s *Source) takeOver(ctx context.Context, r renderedSeg, played float64) []byte {
	v, err := s.voiceFor(r.seg.Role)
	if err != nil {
		return nil // keep the broadcast alive; the next segment re-resolves
	}
	// Re-substitute VoiceToken against the INCOMING voice: a sign-off rendered
	// as "This is Alpha signing off." must not be handed to Bravo to read.
	rest := Remainder(s.spoken(r.seg.Text, v), played)
	if rest == "" {
		return nil
	}
	// The introduction is spoken only if the voice on air is actually changing
	// AND this segment does not already name its speaker. A recast can leave
	// several look-ahead segments voiced by the outgoing correspondent; the
	// first switches with an introduction, the rest simply switch — nobody
	// introduces themselves twice.
	spoken := rest
	if changing := s.changingVoice(v); changing && !r.seg.SelfIntro {
		spoken = s.handoffLine(spokenName(r.voice), spokenName(v)) + " " + rest
	}
	mono, err := s.say(ctx, v, spoken)
	if err != nil {
		s.onSeg(Segment{Key: "handoff:failed", Text: spoken, Role: r.seg.Role}, 0)
		return nil
	}
	pcm := monoToStereo(mono)
	s.onSeg(Segment{Key: r.seg.Key, Text: spoken, Role: r.seg.Role}, s.duration(pcm))
	return pcm
}

// fail records why the broadcast is ending early and returns false for the
// caller's convenience. Only a SEGMENT render reaches here.
func (s *Source) fail(err error) bool {
	s.mu.Lock()
	if s.err == nil {
		s.err = fmt.Errorf("voice cannot render: %w", err)
	}
	s.mu.Unlock()
	return false
}

// Remainder is the part of text still to be spoken when frac (0..1) of its
// audio has played — from the next word boundary; "" when fewer than three
// words are left (not worth a hand-over).
func Remainder(text string, frac float64) string {
	words := strings.Fields(text)
	skip := int(max(0, min(1, frac)) * float64(len(words)))
	if len(words)-skip < 3 {
		return ""
	}
	return strings.Join(words[skip:], " ")
}

// spoken resolves VoiceToken in a segment's TEXT to the reading voice's name.
func (s *Source) spoken(text string, v Voice) string {
	return strings.ReplaceAll(text, VoiceToken, spokenName(v))
}

// cacheKey identifies rendered audio: what was said, and BY WHOM.
//
// The voice's FULL name, not its spoken one: two engines can share a spoken
// name — a macOS "Daniel" and a Piper key that renders as "Daniel" — and their
// audio is not interchangeable.
func cacheKey(segKey string, v Voice) string {
	name := ""
	if v != nil {
		name = v.Name()
	}
	return strings.ReplaceAll(segKey, VoiceToken, name) + "\x00" + name
}

// render voices a segment as mono PCM, cached by (segment key, voice). It also
// reports the soft generation in force when it rendered, so the writer can tell
// a stale segment from a current one without re-resolving.
func (s *Source) render(ctx context.Context, seg Segment, v Voice) ([]byte, uint64, error) {
	pcm, err := s.renderText(ctx, cacheKey(seg.Key, v), v, s.spoken(seg.Text, v))
	soft, _ := s.generations()
	return pcm, soft, err
}

// renderText is the one place audio is synthesised and cached. Segments and
// hand-over lines both come through it, so the cache bound covers everything
// the Source holds.
//
// The result is stored UNCONDITIONALLY. An earlier design refused to cache a
// render that straddled a voice change, on the theory that it might be stale —
// but the key already names the voice, so it cannot be: audio keyed to the
// voice that produced it stays correct forever, whoever is reading now.
func (s *Source) renderText(ctx context.Context, key string, v Voice, text string) ([]byte, error) {
	s.mu.Lock()
	if pcm, ok := s.cache[key]; ok {
		s.mu.Unlock()
		return pcm, nil
	}
	s.mu.Unlock()

	pcm, err := s.say(ctx, v, text)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.cache[key]; exists {
		return pcm, nil // another goroutine won the race; the byte count must not double-count
	}
	s.cache[key] = pcm
	s.order = append(s.order, key)
	s.bytes += len(pcm)
	// Oldest-first eviction, counter-bounded (P10-02): each pass removes one
	// entry, and there cannot be more passes than there are entries.
	for i := 0; i < len(s.order) && s.bytes > maxCachedBytes && len(s.order) > 1; i++ {
		oldest := s.order[0]
		s.bytes -= len(s.cache[oldest])
		delete(s.cache, oldest)
		s.order = s.order[1:]
	}
	return pcm, nil
}

// say voices text with v (uncached) as MONO PCM; callers widen at write time —
// the cache holds mono (0.9.0 exit measurement: 64 cached stereo segments crept
// RSS by ~35 MB in 90 s).
func (s *Source) say(ctx context.Context, v Voice, text string) ([]byte, error) {
	if v == nil {
		return nil, ErrNoVoice
	}
	return v.Say(ctx, Pronounce(text)) // voice-only spellings; the marquee shows the text
}

// Cached reports the rendered-audio cache's size — entries and mono PCM bytes —
// for the diagnostic dump. O(1): the byte count is maintained on insert and
// eviction rather than summed on every call, because the gauge is read on the
// update loop.
func (s *Source) Cached() (entries int, bytes int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.cache), s.bytes
}

// maxCachedBytes bounds the rendered-audio cache in BYTES, not entries.
//
// Entries stopped being a usable bound once a cycle can hold more than one
// voice: 40 entries in one voice is ~29 MB, but the same 40 entries across two
// correspondents plus their hand-over lines is not, and the number that matters
// to a small Linux box is megabytes. 40 MB covers a cycle in one voice (~29 MB),
// a second correspondent's sections, and ≤ 4.6 MB of hand-over lines.
const maxCachedBytes = 40 << 20

func (s *Source) silence(d time.Duration) []byte {
	return make([]byte, int(d.Seconds()*float64(s.Rate()))*4)
}

// monoToStereo duplicates each 16-bit sample into both channels.
func monoToStereo(mono []byte) []byte {
	out := make([]byte, 0, len(mono)*2)
	for i := 0; i+2 <= len(mono); i += 2 {
		v := binary.LittleEndian.Uint16(mono[i:])
		out = binary.LittleEndian.AppendUint16(out, v)
		out = binary.LittleEndian.AppendUint16(out, v)
	}
	return out
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
