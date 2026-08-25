package synth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Voice renders narration to 16-bit little-endian MONO PCM at Rate().
// Adapters exec via argv slices only — never a shell string — and the
// narration travels by stdin or a 0600 temp file, never inside an argv
// element (§10.5). Product text is untrusted input.
type Voice interface {
	Name() string
	Rate() int
	Say(ctx context.Context, text string) ([]byte, error)
}

// ErrNoVoice reports that no engine is available on this host.
var ErrNoVoice = errors.New("synth: no voice engine available")

// hostileText guards the §10.5 contract in tests and at the boundary: the
// narration is never placed in argv, so these characters are inert.
func writeNarration(dir, text string) (string, error) {
	f, err := os.CreateTemp(dir, "watchpost-narration-*.txt")
	if err != nil {
		return "", err
	}
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return "", err
	}
	if _, err := f.WriteString(text); err != nil {
		_ = f.Close()
		return "", err
	}
	return f.Name(), f.Close()
}

// --- macOS: the built-in `say` ---

// SayVoice is Apple's built-in synthesizer (darwin only). Narration goes
// through a temp file (-f); output is a WAV at 22.05 kHz mono. Voice ""
// means the system's selected voice.
type SayVoice struct {
	Voice string // e.g. "Samantha"; "" = system default
}

// Name implements Voice: the correspondent's name as spoken in the tail.
// The system voice cannot be named by `say` on modern macOS (a Siri
// voice), so it introduces itself as such (UAT 88).
func (v SayVoice) Name() string {
	if v.Voice != "" {
		return v.Voice
	}
	if name := systemVoiceName(); name != "" {
		return name
	}
	return "the System Voice"
}

// systemVoiceName reads macOS's selected voice (argv only; best effort) —
// once per process (red-team 0.9.0 C-5): Name() is called on the update
// loop and under the Source lock, so it must not spawn a subprocess.
func systemVoiceName() string {
	systemVoiceOnce.Do(func() {
		out, err := exec.Command("defaults", "read", "com.apple.speech.voice.prefs", "SelectedVoiceName").Output()
		if err == nil {
			systemVoiceMemo = strings.TrimSpace(string(out))
		}
	})
	return systemVoiceMemo
}

// The memo for systemVoiceName: the selection does not change under a
// running process in any way the app could follow.
var (
	systemVoiceOnce sync.Once
	systemVoiceMemo string
)

// Rate implements Voice.
func (SayVoice) Rate() int { return 22050 }

// Say implements Voice.
func (v SayVoice) Say(ctx context.Context, text string) ([]byte, error) {
	if runtime.GOOS != "darwin" {
		return nil, ErrNoVoice
	}
	dir, err := os.MkdirTemp("", "watchpost-say-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	in, err := writeNarration(dir, text)
	if err != nil {
		return nil, err
	}
	out := filepath.Join(dir, "out.wav")
	args := []string{"-o", out, "--file-format=WAVE", "--data-format=LEI16@22050", "-f", in}
	if v.Voice != "" {
		args = append([]string{"-v", v.Voice}, args...)
	}
	cmd := exec.CommandContext(ctx, "say", args...)
	if b, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("say: %w: %s", err, strings.TrimSpace(string(b)))
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		return nil, err
	}
	return wavPCM(raw)
}

// wavPCM strips a RIFF/WAVE header, returning the data chunk.
func wavPCM(raw []byte) ([]byte, error) {
	if err := invariant.Check(len(raw) > 12 && string(raw[:4]) == "RIFF" && string(raw[8:12]) == "WAVE", "synth: not a WAVE file"); err != nil {
		return nil, err
	}
	for off := 12; off+8 <= len(raw); {
		id := string(raw[off : off+4])
		size := int(raw[off+4]) | int(raw[off+5])<<8 | int(raw[off+6])<<16 | int(raw[off+7])<<24
		if id == "data" {
			end := min(len(raw), off+8+size)
			return raw[off+8 : end], nil
		}
		off += 8 + size + size%2
	}
	return nil, errors.New("synth: WAVE file has no data chunk")
}

// --- Piper (Linux / Windows; any host with a Piper install) ---

// PiperVoice runs a Piper install: `piper --model <onnx> --output-raw`,
// narration on stdin, raw 16-bit mono PCM on stdout at the model's rate.
type PiperVoice struct {
	Install Install // from EnsurePiper / FindPiper
}

// Name implements Voice: the catalogue name, else the voice's given name
// from the model file ("en_US-lessac-medium" -> "Lessac").
func (p PiperVoice) Name() string {
	if p.Install.Voice != "" {
		return p.Install.Voice
	}
	parts := strings.Split(strings.TrimSuffix(filepath.Base(p.Install.Model), ".onnx"), "-")
	if len(parts) < 2 || parts[1] == "" {
		return "Piper"
	}
	return strings.ToUpper(parts[1][:1]) + parts[1][1:]
}

// Rate implements Voice.
func (p PiperVoice) Rate() int { return p.Install.SampleRate }

// withLibPath is the process environment with LD_LIBRARY_PATH set to dir —
// replacing, not duplicating, any existing value (red-team 0.9.0 S-F8:
// glibc honours the first occurrence, so a user's own setting used to win
// and Piper failed to load its bundled onnxruntime).
func withLibPath(dir string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "LD_LIBRARY_PATH=") {
			env = append(env, kv)
		}
	}
	return append(env, "LD_LIBRARY_PATH="+dir)
}

// Say implements Voice.
func (p PiperVoice) Say(ctx context.Context, text string) ([]byte, error) {
	if p.Install.Binary == "" || p.Install.Model == "" {
		return nil, ErrNoVoice
	}
	cmd := exec.CommandContext(ctx, p.Install.Binary, "--model", p.Install.Model, "--output-raw")
	cmd.Dir = filepath.Dir(p.Install.Binary)
	cmd.Env = withLibPath(filepath.Dir(p.Install.Binary)) // the archive's own libraries, on any host
	cmd.Stdin = strings.NewReader(text + "\n")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("piper: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// TextTicker is the degradation voice: it renders nothing and reports so
// the UI can show the narration as text (C-6″).
type TextTicker struct{}

// Name implements Voice.
func (TextTicker) Name() string { return "text" }

// Rate implements Voice.
func (TextTicker) Rate() int { return 22050 }

// Say implements Voice.
func (TextTicker) Say(context.Context, string) ([]byte, error) { return nil, ErrNoVoice }
