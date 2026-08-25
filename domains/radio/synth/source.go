package synth

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// VoiceToken in a segment's text is spoken (and shown) as the CURRENT
// voice's name — so a sign-off planned in one voice is read correctly by
// the voice that reaches it (UAT 94).
const VoiceToken = "{{voice}}"

// Source is the synthesized broadcast as a PCM stream (16-bit LE STEREO at
// the voice's rate): it narrates one cycle, then — when Repeat is on —
// asks for the next cycle's segments (fresh products) and continues; a
// newly issued product never swaps mid-segment (§10.4). Rendered audio is
// cached by segment key. The voice can change mid-broadcast (UAT 94):
// playback hands over at the spot reached, without restarting.
type Source struct {
	next  func(ctx context.Context) ([]Segment, error) // the next cycle's segments
	onSeg func(Segment, time.Duration)                 // narration text + its spoken length, for the marquee
	gap   time.Duration

	mu     sync.Mutex
	voice  Voice
	prev   string            // the voice handed over from (for the hand-over line)
	gen    uint64            // bumps on every SetVoice; audio rendered under an older gen is re-voiced before it plays
	repeat bool              // loop cycles (UAT 83: [r] Repeat); off = one cycle, then the stream ends
	cache  map[string][]byte // segment key -> rendered stereo PCM (this voice's)
	order  []string
	err    error // why the broadcast ended early (a voice that cannot render); nil = it ran to its end
}

// Err is why the stream ended before its cycle did — a voice that cannot
// render (uninstalled, broken) — or nil for a natural end. The deck reads
// it when the engine reports the end (red-team 0.9.0 C-4): a render
// failure must never pass for "broadcast complete", which under Repeat:
// Watchlist would spin through every favourite.
func (s *Source) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// NewSource builds a source; onSeg may be nil.
func NewSource(voice Voice, next func(context.Context) ([]Segment, error), onSeg func(Segment, time.Duration)) (*Source, error) {
	if err := invariant.Check(voice != nil && next != nil, "synth: voice and segment provider are required"); err != nil {
		return nil, err
	}
	if onSeg == nil {
		onSeg = func(Segment, time.Duration) {}
	}
	return &Source{voice: voice, next: next, onSeg: onSeg, gap: 400 * time.Millisecond, cache: map[string][]byte{}}, nil
}

// Rate is the PCM rate (the voice's — fixed for the stream's life).
func (s *Source) Rate() int {
	v, _ := s.current()
	return v.Rate()
}

// current is the voice in force and its generation.
func (s *Source) current() (Voice, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.voice, s.gen
}

// SetVoice hands the running broadcast to another voice (UAT 94): the
// segment playing resumes from the spot reached, segments rendered ahead
// are re-voiced, and the cache starts over. Refused when the voice speaks
// at another sample rate — the stream's rate was fixed at start, so the
// caller re-tunes instead.
func (s *Source) SetVoice(v Voice) error {
	if err := invariant.Check(v != nil, "synth: a voice is required"); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if v.Rate() != s.voice.Rate() {
		return fmt.Errorf("synth: %s speaks at %d Hz; this broadcast runs at %d Hz", v.Name(), v.Rate(), s.voice.Rate())
	}
	if v.Name() == s.voice.Name() {
		return nil // the same correspondent: nothing to hand over
	}
	s.prev, s.voice = s.voice.Name(), v
	s.gen++
	s.cache, s.order = map[string][]byte{}, nil
	return nil
}

// Handoff is the line the new voice opens with (HUM LEAD, UAT 94): it
// covers the remainder's render time and makes the change deliberate.
func Handoff(newName, oldName string) string {
	return fmt.Sprintf("This is %s, taking over for %s.", newName, oldName)
}

// handoff is the current hand-over line.
func (s *Source) handoff() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Handoff(s.voice.Name(), s.prev)
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

// duration is the spoken length of stereo PCM at the voice's rate.
func (s *Source) duration(pcm []byte) time.Duration {
	return time.Duration(float64(len(pcm)/4) / float64(s.Rate()) * float64(time.Second))
}

// Open returns a reader that yields the broadcast until ctx ends (or the
// cycle ends with Repeat off). Segments are rendered one ahead of playback
// (UAT 81: no pause between segments), and cancelling ctx unblocks any
// pending write at once (fast stop).
func (s *Source) Open(ctx context.Context) io.Reader {
	pr, pw := io.Pipe()
	rendered := make(chan renderedSeg, 1) // one segment of look-ahead
	go s.renderLoop(ctx, rendered)
	go func() {
		defer func() { _ = pw.Close() }()
		for r := range rendered {
			if !s.play(ctx, pw, r) {
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

type renderedSeg struct {
	seg Segment
	pcm []byte
	gen uint64 // the voice generation that rendered it
}

// renderLoop plans cycles and renders segments ahead of playback.
func (s *Source) renderLoop(ctx context.Context, out chan<- renderedSeg) {
	defer close(out)
	// Bounded per P10-02: each cycle re-plans; the loop ends with ctx, or
	// after one cycle when Repeat is off.
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
			pcm, gen, err := s.render(ctx, seg)
			if err != nil {
				s.fail(err)
				return // the stream ends; Err() tells the deck it was not a completion
			}
			select {
			case out <- renderedSeg{seg: seg, pcm: pcm, gen: gen}:
			case <-ctx.Done():
				return
			}
		}
		if !s.repeating() {
			return // one broadcast, then the stream ends (UAT 83)
		}
	}
}

// play streams one rendered segment to the pipe in 100 ms chunks, then the
// inter-segment gap. Between chunks it watches for a voice change (UAT
// 94): the new voice opens with the hand-over line while the remainder of
// the current line — from the word reached — renders alongside, then
// playback continues there. A segment rendered ahead by an older voice is
// re-voiced whole before it plays. False when the pipe is gone (stop) or
// the voice cannot render.
func (s *Source) play(ctx context.Context, pw *io.PipeWriter, r renderedSeg) bool {
	_, gen := s.current()
	pcm := r.pcm
	if r.gen != gen {
		var err error
		if pcm, gen, err = s.render(ctx, r.seg); err != nil { // gen: what actually voiced it — a change since hands over below
			return s.fail(err)
		}
	}
	pcm = monoToStereo(pcm) // the cache and the channel carry mono; the pipe is stereo
	text := s.spoken(r.seg.Text)
	s.onSeg(Segment{Key: r.seg.Key, Text: text}, s.duration(pcm))
	chunk := s.Rate() / 10 * 4
	// Bounded: one chunk per pass, plus one hand-over per voice change.
	for written := 0; written < len(pcm); {
		if _, g := s.current(); g != gen {
			gen = g
			rest := Remainder(text, float64(written)/float64(len(pcm)))
			if rest == "" {
				break // too little left to hand over: the next segment follows
			}
			next, ok := s.handOver(ctx, pw, r.seg.Key, rest)
			if !ok {
				return false // handOver recorded any render failure
			}
			text, pcm, written = rest, next, 0
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

// handOver speaks the hand-over line in the new voice while rest renders
// in the background, and returns rest's audio, ready to continue with.
func (s *Source) handOver(ctx context.Context, pw *io.PipeWriter, key, rest string) ([]byte, bool) {
	type rendered struct {
		pcm []byte
		err error
	}
	ahead := make(chan rendered, 1)
	v, _ := s.current()
	go func() {
		pcm, err := s.say(ctx, v, rest)
		ahead <- rendered{pcm, err}
	}()
	line := s.handoff()
	if mono, err := s.say(ctx, v, line); err == nil {
		pcm := monoToStereo(mono)
		s.onSeg(Segment{Key: key, Text: line}, s.duration(pcm))
		if _, err := pw.Write(pcm); err != nil {
			return nil, false
		}
	}
	next := <-ahead
	if next.err != nil {
		return nil, s.fail(next.err)
	}
	pcm := monoToStereo(next.pcm)
	s.onSeg(Segment{Key: key, Text: rest}, s.duration(pcm))
	return pcm, true
}

// fail records why the broadcast is ending early (every render path —
// render-ahead, re-voice, hand-over — reports through here, round 2 N-4)
// and returns false for the caller's convenience.
func (s *Source) fail(err error) bool {
	s.mu.Lock()
	if s.err == nil {
		s.err = fmt.Errorf("voice cannot render: %w", err)
	}
	s.mu.Unlock()
	return false
}

// Remainder is the part of text still to be spoken when frac (0..1) of its
// audio has played — from the next word boundary; "" when fewer than
// three words are left (not worth a hand-over).
func Remainder(text string, frac float64) string {
	words := strings.Fields(text)
	skip := int(max(0, min(1, frac)) * float64(len(words)))
	if len(words)-skip < 3 {
		return ""
	}
	return strings.Join(words[skip:], " ")
}

// spoken resolves VoiceToken to the current voice's name.
func (s *Source) spoken(text string) string {
	v, _ := s.current()
	return strings.ReplaceAll(text, VoiceToken, v.Name())
}

// render voices a segment as mono PCM (cached by key; play widens it). It
// returns the generation that voiced it: the voice and generation are
// captured up front, and the audio is cached only if no SetVoice happened
// meanwhile — a render that straddles a change must never land in the new
// voice's cache (it would play in the old voice later).
func (s *Source) render(ctx context.Context, seg Segment) ([]byte, uint64, error) {
	s.mu.Lock()
	if pcm, ok := s.cache[seg.Key]; ok {
		s.mu.Unlock()
		return pcm, s.gen, nil
	}
	v, gen := s.voice, s.gen
	s.mu.Unlock()
	pcm, err := s.say(ctx, v, strings.ReplaceAll(seg.Text, VoiceToken, v.Name()))
	if err != nil {
		return nil, gen, err
	}
	s.mu.Lock()
	if s.gen == gen {
		s.cache[seg.Key] = pcm
		s.order = append(s.order, seg.Key)
		if len(s.order) > maxCached { // bound the audio cache to about a cycle
			delete(s.cache, s.order[0])
			s.order = s.order[1:]
		}
	}
	s.mu.Unlock()
	return pcm, gen, nil
}

// say voices text with v (uncached) as MONO PCM; callers widen at write
// time — the cache holds mono (0.9.0 exit measurement: 64 cached stereo
// segments crept RSS by ~35 MB in 90 s).
func (s *Source) say(ctx context.Context, v Voice, text string) ([]byte, error) {
	return v.Say(ctx, Pronounce(text)) // voice-only spellings; the marquee shows the text
}

// maxCached bounds the rendered-audio cache: a cycle is 10–20 segments and
// only Repeat replays them; 24 mono segments is ~16 MB at the worst.
const maxCached = 40

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
