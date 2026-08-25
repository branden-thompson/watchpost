package synth

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Piper install (B4 step 2, HUM LEAD: "watchpost will install piper for
// users on first run"). Every artifact is pinned by SHA-256 here; the
// download is verified before anything is extracted or executed. macOS is
// not in the manifest: the upstream macOS archive ships without its
// libraries and cannot run — macOS uses the built-in `say` instead.

// Install locates a usable Piper.
type Install struct {
	Binary     string // piper executable
	Model      string // voice .onnx (the .onnx.json sits beside it)
	SampleRate int    // model sample rate
}

// Asset is one pinned download.
type Asset struct {
	URL    string
	SHA256 string
	Size   int64
}

// piperRelease is the pinned upstream release.
const piperRelease = "2023.11.14-2"

// piperAssets maps GOOS/GOARCH to the release archive (checksums computed
// from the artifacts on 2026-08-24).
func piperAssets() map[string]Asset {
	base := "https://github.com/rhasspy/piper/releases/download/" + piperRelease + "/"
	return map[string]Asset{
		"linux/amd64":   {base + "piper_linux_x86_64.tar.gz", "a50cb45f355b7af1f6d758c1b360717877ba0a398cc8cbe6d2a7a3a26e225992", 26460462},
		"linux/arm64":   {base + "piper_linux_aarch64.tar.gz", "fea0fd2d87c54dbc7078d0f878289f404bd4d6eea6e7444a77835d1537ab88eb", 26004717},
		"linux/arm":     {base + "piper_linux_armv7l.tar.gz", "c6946fcd57c705ed1d4666ea880f80ba0bbbd14de62ecbdd13460baf3bac8e37", 25445955},
		"windows/amd64": {base + "piper_windows_amd64.zip", "f3c58906402b24f3a96d92145f58acba6d86c9b5db896d207f78dc80811efcea", 22477236},
	}
}

// Voice model: en_US-lessac-medium (MIT-licensed voice; 22.05 kHz).
const (
	voiceName = "en_US-lessac-medium"
	voiceRate = 22050
)

func voiceAssets() [2]Asset {
	base := "https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/lessac/medium/"
	return [2]Asset{
		{base + voiceName + ".onnx", "5efe09e69902187827af646e1a6e9d269dee769f9877d17b16b1b46eeaaf019f", 63201294},
		{base + voiceName + ".onnx.json", "efe19c417bed055f2d69908248c6ba650fa135bc868b0e6abb3da181dab690a0", 0},
	}
}

// Progress reports download progress: which artifact, bytes so far, total.
type Progress func(what string, done, total int64)

// PiperSupported reports whether this host is in the manifest.
func PiperSupported() bool {
	_, ok := piperAssets()[runtime.GOOS+"/"+runtime.GOARCH]
	return ok
}

// FindPiper returns an existing install under dir, if complete.
func FindPiper(dir string) (Install, bool) {
	if dir == "" || !filepath.IsAbs(dir) {
		return Install{}, false // never a CWD-relative binary (red-team 0.9.0 S-F5: $HOME unset must not exec ./piper/piper)
	}
	bin := filepath.Join(dir, "piper", "piper")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	model := filepath.Join(dir, "voices", voiceName+".onnx")
	for _, p := range []string{bin, model, model + ".json"} {
		if _, err := os.Stat(p); err != nil {
			return Install{}, false
		}
	}
	return Install{Binary: bin, Model: model, SampleRate: voiceRate}, true
}

// EnsurePiper installs Piper and the voice under dir when missing,
// verifying every artifact against the manifest; progress may be nil.
// installMu serializes EnsurePiper (one download at a time, C-9).
var installMu sync.Mutex

func EnsurePiper(ctx context.Context, dir, userAgent string, progress Progress) (Install, error) {
	if dir == "" || !filepath.IsAbs(dir) {
		return Install{}, fmt.Errorf("synth: no cache directory for the voice (set XDG_CACHE_HOME or HOME to an absolute path)") // round 2 N-5: never download into a relative dir FindPiper will refuse
	}
	if inst, ok := FindPiper(dir); ok {
		return inst, nil
	}
	// One install at a time (red-team 0.9.0 C-9): two overlapping tune-ins
	// on a fresh host must not write the same .part file.
	installMu.Lock()
	defer installMu.Unlock()
	if inst, ok := FindPiper(dir); ok {
		return inst, nil // the other caller finished it
	}
	asset, ok := piperAssets()[runtime.GOOS+"/"+runtime.GOARCH]
	if !ok {
		return Install{}, fmt.Errorf("synth: no Piper build for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if progress == nil {
		progress = func(string, int64, int64) {}
	}
	if err := installArtifacts(ctx, dir, asset, userAgent, progress); err != nil {
		return Install{}, err
	}
	inst, ok := FindPiper(dir)
	if !ok {
		return Install{}, errors.New("synth: Piper archive did not contain the expected files")
	}
	return inst, nil
}

// installArtifacts fetches and unpacks the Piper archive and the voice
// files under dir (split from EnsurePiper, P10-04); every artifact is
// verified by download before extract touches it.
func installArtifacts(ctx context.Context, dir string, asset Asset, userAgent string, progress Progress) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, stale := range []string{filepath.Join(dir, "piper-archive.part"), filepath.Join(dir, "voices", voiceName+".onnx.part"), filepath.Join(dir, "voices", voiceName+".onnx.json.part")} {
		_ = os.Remove(stale) // an interrupted earlier install (Ctrl-C) leaves nothing behind (Linux F8)
	}
	archive, err := download(ctx, asset, userAgent, filepath.Join(dir, "piper-archive"), "Piper", progress)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(archive) }()
	if err := extract(archive, dir); err != nil {
		return err
	}
	voices := filepath.Join(dir, "voices")
	if err := os.MkdirAll(voices, 0o755); err != nil {
		return err
	}
	for i, a := range voiceAssets() {
		name := voiceName + ".onnx"
		if i == 1 {
			name += ".json"
		}
		if _, err := download(ctx, a, userAgent, filepath.Join(voices, name), "voice", progress); err != nil {
			return err
		}
	}
	return nil
}

// download streams an asset to dest (via a temp file) while hashing;
// a checksum mismatch removes the file and returns an error.
func download(ctx context.Context, a Asset, userAgent, dest, what string, progress Progress) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	client := &http.Client{Timeout: 10 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		// Pinned artifacts travel over HTTPS only, redirects included (red-team 0.9.0 S-F4).
		if req.URL.Scheme != "https" {
			return fmt.Errorf("synth: refusing a redirect to %s://%s — artifacts are fetched over https only", req.URL.Scheme, req.URL.Host)
		}
		if len(via) >= 10 {
			return errors.New("synth: too many redirects")
		}
		return nil
	}}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: HTTP %d from %s", what, resp.StatusCode, a.URL)
	}
	total := a.Size
	if total == 0 {
		total = resp.ContentLength
	}
	tmp := dest + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	limit := a.Size + 1 // the manifest knows the size: a mirror streaming garbage stops here, not at disk-full (C-8)
	if a.Size == 0 {
		limit = 1 << 20 // an unsized asset is a small JSON sidecar; 1 MB is generous
	}
	body := io.LimitReader(resp.Body, limit)
	n, err := io.Copy(io.MultiWriter(f, h, &progressWriter{what: what, total: total, progress: progress}), body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && a.Size > 0 && n < a.Size {
		err = fmt.Errorf("%s: download incomplete (got %d of %d bytes) — check the connection and retry", what, n, a.Size) // round 2 N-8: not "tampered"
	}
	if err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != a.SHA256 {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("%s: checksum mismatch for %s (got %s) — download refused", what, filepath.Base(a.URL), got[:12])
	}
	return dest, os.Rename(tmp, dest)
}

// progressWriter reports bytes as they pass through io.Copy.
type progressWriter struct {
	what     string
	done     int64
	total    int64
	progress Progress
}

func (w *progressWriter) Write(p []byte) (int, error) {
	w.done += int64(len(p))
	w.progress(w.what, w.done, w.total)
	return len(p), nil
}

// extract unpacks a .tar.gz or .zip into dir, refusing paths that escape it.
func extract(archive, dir string) error {
	if strings.HasSuffix(archive, ".zip") || isZip(archive) {
		return extractZip(archive, dir)
	}
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for i := 0; i < 10000; i++ { // bounded per P10-02: archives hold a few hundred entries
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := tarEntry(dir, hdr, tr); err != nil {
			return err
		}
	}
	return errors.New("synth: archive too large")
}

func isZip(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	var sig [4]byte
	_, _ = io.ReadFull(f, sig[:])
	return sig == [4]byte{'P', 'K', 3, 4}
}

// tarEntry unpacks one tar entry under dir (split from extract, P10-04):
// directories, regular files, and symlinks that stay inside the install
// dir — an absolute or parent-relative link target is refused (red-team
// 0.9.0 S-F3: a link out, then a file written through it, would escape).
func tarEntry(dir string, hdr *tar.Header, tr *tar.Reader) error {
	dst, err := safeJoin(dir, hdr.Name)
	if err != nil {
		return err
	}
	switch hdr.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(dst, 0o755)
	case tar.TypeReg:
		return writeFile(dst, tr, os.FileMode(hdr.Mode))
	case tar.TypeSymlink:
		if filepath.IsAbs(hdr.Linkname) || strings.Contains(hdr.Linkname, "..") {
			return fmt.Errorf("synth: archive entry %s links outside the install dir — refused", hdr.Name)
		}
		if _, err := safeJoin(filepath.Dir(dst), hdr.Linkname); err == nil {
			_ = os.Remove(dst)
			_ = os.Symlink(hdr.Linkname, dst)
		}
	}
	return nil
}

func extractZip(archive, dir string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer func() { _ = zr.Close() }()
	for _, zf := range zr.File {
		dst, err := safeJoin(dir, zf.Name)
		if err != nil {
			return err
		}
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		err = writeFile(dst, rc, zf.Mode())
		_ = rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func writeFile(dst string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode|0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// safeJoin refuses archive entries that would land outside dir.
func safeJoin(dir, name string) (string, error) {
	dst := filepath.Join(dir, filepath.FromSlash(name))
	if !strings.HasPrefix(dst, filepath.Clean(dir)+string(os.PathSeparator)) && dst != filepath.Clean(dir) {
		return "", fmt.Errorf("synth: archive entry escapes the install dir: %q", name)
	}
	return dst, nil
}
