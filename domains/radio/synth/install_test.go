package synth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVoiceCatalogAndInstalledLookup(t *testing.T) {
	// UAT 118: six curated Piper voices, Lessac first (the default); lookup
	// by chooser name or file key, case-insensitive; FindPiperVoice needs
	// the binary and that voice's two files; InstalledVoices lists what is
	// present in catalogue order; FindPiper answers with the first present.
	cat := VoiceCatalog()
	if len(cat) != 6 || cat[0].Name != "Lessac" || DefaultVoice().Key != "en_US-lessac-medium" {
		t.Fatalf("catalogue: %+v", cat)
	}
	seen := map[string]bool{}
	for _, v := range cat {
		if v.Model.SHA256 == "" || len(v.Model.SHA256) != 64 || v.JSON.SHA256 == "" || v.Model.Size == 0 || v.Rate != 22050 || seen[v.Model.SHA256] {
			t.Fatalf("voice %s must carry distinct pinned checksums and a size: %+v", v.Key, v)
		}
		seen[v.Model.SHA256] = true
	}
	if v, ok := VoiceByName("amy"); !ok || v.Key != "en_US-amy-medium" {
		t.Fatal("lookup by name")
	}
	if v, ok := VoiceByName("EN_GB-ALAN-MEDIUM"); !ok || v.Name != "Alan" {
		t.Fatal("lookup by key")
	}
	if _, ok := VoiceByName("System Voice"); ok {
		t.Fatal("a Mac voice is not a catalogue voice")
	}

	dir := t.TempDir()
	if _, ok := FindPiper(dir); ok || len(InstalledVoices(dir)) != 0 {
		t.Fatal("empty dir: nothing installed")
	}
	if _, ok := FindPiperVoice("relative/dir", cat[0]); ok {
		t.Fatal("a relative dir is refused (S-F5)")
	}
	touch := func(rel string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	touch(filepath.Join("voices", "en_US-amy-medium.onnx"))
	touch(filepath.Join("voices", "en_US-amy-medium.onnx.json"))
	if _, ok := FindPiperVoice(dir, cat[1]); ok {
		t.Fatal("voice files without the binary are not an install")
	}
	touch(piperBinary(dir)[len(dir)+1:])
	inst, ok := FindPiperVoice(dir, cat[1])
	if !ok || inst.Voice != "Amy" || inst.SampleRate != 22050 || (PiperVoice{Install: inst}).Name() != "Amy" {
		t.Fatalf("Amy installed: %v %+v", ok, inst)
	}
	if first, ok := FindPiper(dir); !ok || first.Voice != "Amy" {
		t.Fatal("FindPiper answers with the first installed voice")
	}
	if got := InstalledVoices(dir); len(got) != 1 || got[0].Name != "Amy" {
		t.Fatalf("installed: %+v", got)
	}
	if _, ok := FindPiperVoice(dir, cat[0]); ok {
		t.Fatal("Lessac is not installed here")
	}
}
