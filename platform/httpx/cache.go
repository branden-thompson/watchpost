package httpx

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// cache is the URL-keyed response store: a memory tier that every Client
// has, and an optional disk tier (CacheDir) holding the same entries so a
// relaunch is warm. Disk files are a one-line JSON header (redacted URL,
// expiry, validators) followed by the raw body — inspectable, and read
// back without decoding (UAT 73: the earlier base64 format cost a third
// more bytes and a full copy per read). Only public weather data ever
// lands here.
//
// Quality pass Q1 (plan §2.2) added the rules that keep the tiers bounded
// over weeks: a persistence floor (short-lived entries never touch disk),
// one retention rule (an expired entry that carries validators is kept for
// a grace so Q5 can revalidate it), an allow-list sweep of the directory,
// and a cap on the negative cache.
type cache struct {
	dir          string
	maxDiskBytes int64         // directory cap enforced by the sweep
	writes       chan entry    // disk writes happen on one goroutine, off the request path (UAT 73)
	reads        chan struct{} // bounds concurrent disk reads (UAT 74): a 200-goroutine warm launch once spawned ~90 OS threads on short file syscalls
	now          func() time.Time

	mu         sync.Mutex
	mem        map[string]*entry
	bytes      int
	tick       uint64 // LRU clock
	neg        map[string]negEntry
	diskWrites int64 // files written by the writer (the Q1 gate's write-rate counter)
	lastSweep  time.Time

	queued, handled atomic.Int64 // items given to the writer / items the writer has finished (flush waits for equality)
	sweepMu         sync.Mutex   // serialises sweeps
}

// entry is one cached response.
type entry struct {
	URL          string    `json:"url"` // redacted, for humans reading the file
	Expires      time.Time `json:"expires"`
	ETag         string    `json:"etag,omitempty"`          // validators (Q1 stores, Q5 sends): two fixed, bounded fields, never the header map (red-team IS-6)
	LastModified string    `json:"last_modified,omitempty"` //
	Body         []byte    `json:"-"`
	key          string
	used         uint64
	touch        bool // writer: refresh the file's mtime (a 304 renewal), no body
}

// hasValidators reports whether a 304 could renew this entry.
func (e *entry) hasValidators() bool { return e.ETag != "" || e.LastModified != "" }

type negEntry struct {
	err   error
	until time.Time
}

// Memory-tier budget (UAT 73): the 60-location launch holds ~17 MB of raw
// NWS bodies plus 6.6 MB of CO-OPS station lists if unbounded. 8 MB keeps
// the products that are re-read within a cycle (gridpoints shared by two
// consumers, buoy files shared by neighbours); anything larger than a
// quarter of the budget is disk-only — it is parsed once and re-read from
// disk if ever needed. Expired entries are swept first.
const (
	maxMemBytes = 8 << 20
	maxMemEntry = maxMemBytes / 4
	maxEntries  = 4096
	writeQueue  = 256
	maxDiskRead = 4
)

// Bounds added by the quality pass (plan §2.2, §0.8).
const (
	diskFloor        = 5 * time.Minute // caller TTL must exceed this (or pass Persist) for a disk write: obs/alerts never serve a relaunch (L4-F2)
	staleGrace       = 24 * time.Hour  // an expired entry with validators lives this long past Expires, in memory and on disk (CQ-3, PA-4)
	maxNegEntries    = 1024            // negative cache (L1-F11, BQ-9); LRU drop on overflow
	maxDiskBytes     = 256 << 20       // directory cap, oldest by mtime first (L4-F1)
	sweepEvery       = 24 * time.Hour  // plus once at start
	sweepMaxEntries  = 10_000          // listed per pass
	sweepMaxDeletes  = 1_000           // removed per pass
	tmpMaxAge        = time.Hour       // an abandoned temp file older than this is swept
	maxValidatorSize = 512             // bytes per validator field (IS-6)
	headerMaxBytes   = 4096            // the header line the sweep reads
	requiredDirPart  = "watchpost"     // the sweep refuses a directory without this path element (IS-1)
)

// Allow-list (IS-1): the sweep touches only names it wrote. `.json` is the
// pre-UAT-73 format (593 orphans found in DISCOVER, L4-F1).
var (
	cacheNameRe = regexp.MustCompile(`^[0-9a-f]{64}\.(cache|json)$`)
	tmpNameRe   = regexp.MustCompile(`^[0-9a-f]{64}\.cache\.[0-9]+\.tmp$`)
)

func newCache(dir string) *cache { return newCacheWithCap(dir, maxDiskBytes) }

// newCacheWithCap is newCache with the directory cap chosen before the
// writer (and its start sweep) runs — tests use a small cap.
func newCacheWithCap(dir string, capBytes int64) *cache {
	c := &cache{dir: dir, maxDiskBytes: capBytes, now: time.Now, mem: map[string]*entry{}, neg: map[string]negEntry{}, reads: make(chan struct{}, maxDiskRead)}
	if dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil { // private, like the config dir (red-team 0.9.0 S-F10)
			c.dir = "" // no disk tier; memory still works
		}
	}
	if c.dir != "" {
		c.writes = make(chan entry, writeQueue)
		go c.writer()
	}
	return c
}

// get returns a fresh body from memory, else from disk (promoting it when
// it fits the memory budget). A stale memory entry means the disk copy is
// no fresher: the read is skipped (L4-F3 — 45k reads/day of files known
// to be expired).
func (c *cache) get(rawURL string) ([]byte, bool) {
	now := c.now()
	c.mu.Lock()
	if e, ok := c.mem[rawURL]; ok {
		if now.Before(e.Expires) {
			c.tick++
			e.used = c.tick
			c.mu.Unlock()
			return e.Body, true
		}
		c.mu.Unlock()
		return nil, false // stale in memory: the disk file is the same entry or older
	}
	c.mu.Unlock()
	if c.dir == "" {
		return nil, false
	}
	c.reads <- struct{}{}
	e, ok := readEntry(c.path(rawURL))
	<-c.reads
	if !ok || !now.Before(e.Expires) {
		return nil, false
	}
	c.remember(rawURL, e)
	return e.Body, true
}

// validators are the response fields a 304 can renew an entry from.
type validators struct {
	etag, lastModified string
}

// validatorsOf extracts bounded, header-safe validators from a response:
// a value over maxValidatorSize or carrying a control byte is dropped
// (a CR/LF-bearing ETag echoed in If-None-Match is rejected by net/http
// and would fail every revalidation — IS-6).
func validatorsOf(hdr http.Header) validators {
	return validators{etag: safeValidator(hdr.Get("ETag")), lastModified: safeValidator(hdr.Get("Last-Modified"))}
}

func safeValidator(v string) string {
	if len(v) > maxValidatorSize {
		return ""
	}
	for i := 0; i < len(v); i++ {
		if b := v[i]; b < 0x20 || b == 0x7f {
			return ""
		}
	}
	return v
}

// put stores a body in memory (budgeted) and, past the persistence floor
// or when the caller asked to Persist, queues it for disk.
func (c *cache) put(rawURL string, body []byte, expires time.Time, v validators, persist bool) {
	e := entry{URL: RedactURL(rawURL), Expires: expires, ETag: v.etag, LastModified: v.lastModified, Body: body}
	c.remember(rawURL, e)
	if c.writes == nil || (!persist && expires.Sub(c.now()) <= diskFloor) {
		return
	}
	e.key = rawURL
	c.enqueue(e)
}

// enqueue hands an item to the writer; a full queue drops it (the entry
// just misses the warm relaunch) rather than blocking the request path.
func (c *cache) enqueue(e entry) {
	select {
	case c.writes <- e:
		c.queued.Add(1)
	default:
	}
}

// renew extends an entry's life in place after a 304 (Q5 calls it): the
// memory expiry moves, and the disk file's mtime is refreshed on the
// writer goroutine so the sweep's max(Expires, mtime) rule keeps it (R2-9).
func (c *cache) renew(rawURL string, expires time.Time) {
	c.mu.Lock()
	if e, ok := c.mem[rawURL]; ok {
		e.Expires = expires
	}
	c.mu.Unlock()
	if c.writes == nil {
		return
	}
	c.enqueue(entry{key: rawURL, Expires: expires, touch: true})
}

// remember places an entry in the memory tier within the byte budget.
func (c *cache) remember(rawURL string, e entry) {
	if len(e.Body) > maxMemEntry {
		return // disk-only
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if old, ok := c.mem[rawURL]; ok {
		c.bytes -= len(old.Body)
	}
	c.tick++
	e.used = c.tick
	c.mem[rawURL] = &e
	c.bytes += len(e.Body)
	if c.bytes > maxMemBytes || len(c.mem) > maxEntries {
		c.evictLocked()
	}
}

// evictLocked drops expired entries that nothing can renew (no validators,
// or past the grace), then least-recently-used ones — an expired entry
// with validators is an LRU citizen like any other, so a 304 has a body to
// renew (PF-4) — until the tier is within budget (caller holds mu).
func (c *cache) evictLocked() {
	now := c.now()
	for k, e := range c.mem {
		if !now.Before(e.Expires) && (!e.hasValidators() || !now.Before(e.Expires.Add(staleGrace))) {
			c.bytes -= len(e.Body)
			delete(c.mem, k)
		}
	}
	// Bounded per P10-02: at most one eviction per entry present.
	for i, n := 0, len(c.mem); i < n && (c.bytes > maxMemBytes || len(c.mem) > maxEntries); i++ {
		var victim string
		oldest := ^uint64(0)
		for k, e := range c.mem {
			if e.used < oldest {
				victim, oldest = k, e.used
			}
		}
		if victim == "" {
			return
		}
		c.bytes -= len(c.mem[victim].Body)
		delete(c.mem, victim)
	}
}

// writer persists queued entries one at a time (atomic rename), refreshes
// mtimes for renewals, and sweeps the directory at start and daily —
// checked per write, so no timer goroutine and no new loop shape.
func (c *cache) writer() {
	c.sweep(c.now())
	for e := range c.writes {
		if now := c.now(); now.Sub(c.lastSweepTime()) >= sweepEvery {
			c.sweep(now)
		}
		if e.touch {
			now := c.now()
			_ = os.Chtimes(c.path(e.key), now, now)
		} else {
			c.writeEntry(e)
		}
		c.handled.Add(1)
	}
}

func (c *cache) writeEntry(e entry) {
	hdr, err := json.Marshal(e)
	if err != nil {
		return
	}
	path := c.path(e.key)
	// A unique temp name (red-team 0.9.0 F8): `report` and the dashboard
	// share this directory and may write the same entry at once.
	f, err := os.CreateTemp(c.dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return
	}
	tmp := f.Name()
	_, werr := f.Write(append(append(hdr, '\n'), e.Body...))
	if cerr := f.Close(); werr == nil && cerr == nil && os.Chmod(tmp, 0o600) == nil && os.Rename(tmp, path) == nil {
		c.mu.Lock()
		c.diskWrites++
		c.mu.Unlock()
		return
	}
	_ = os.Remove(tmp)
}

func (c *cache) lastSweepTime() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastSweep
}

// sweep is the disk tier's only deleter (IS-1: an allow-list, never a
// deny-list). Non-recursive; regular files only (symlinks and directories
// are never touched); names it wrote: `<sha256>.cache`, the legacy
// `<sha256>.json`, and its own temp files older than tmpMaxAge. A cache
// file goes when max(Expires, mtime) + staleGrace is past; then, if the
// directory exceeds the cap, the oldest by mtime go until it fits. It
// refuses to run at all unless the directory has a "watchpost" path
// element — a misconfigured $XDG_CACHE_HOME must never point it at $HOME.
func (c *cache) sweep(now time.Time) {
	c.sweepMu.Lock() // one sweep at a time: the writer's start sweep and a daily one must never overlap
	defer c.sweepMu.Unlock()
	c.mu.Lock()
	c.lastSweep = now
	c.mu.Unlock()
	if !c.sweepAllowed() {
		return
	}
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}
	var keep []os.FileInfo
	deletes := 0
	for i, de := range entries {
		if i >= sweepMaxEntries || deletes >= sweepMaxDeletes {
			break
		}
		if !de.Type().IsRegular() { // ReadDir types come from lstat: a symlink is not regular
			continue
		}
		info, err := de.Info()
		if err != nil {
			continue
		}
		switch {
		case tmpNameRe.MatchString(de.Name()):
			if now.Sub(info.ModTime()) > tmpMaxAge && os.Remove(filepath.Join(c.dir, de.Name())) == nil {
				deletes++
			}
		case cacheNameRe.MatchString(de.Name()):
			if c.fileExpired(filepath.Join(c.dir, de.Name()), info.ModTime(), now) {
				if os.Remove(filepath.Join(c.dir, de.Name())) == nil {
					deletes++
				}
				continue
			}
			keep = append(keep, info)
		}
	}
	c.enforceCap(keep, deletes)
}

func (c *cache) sweepAllowed() bool {
	if c.dir == "" {
		return false
	}
	for _, part := range strings.Split(filepath.ToSlash(c.dir), "/") {
		if part == requiredDirPart {
			return true
		}
	}
	return false
}

// fileExpired reads only the header line: a legacy `.json` is always
// expired (its format is unreadable); a `.cache` expires at
// max(Expires, mtime) + staleGrace, so a 304-renewed file (fresh mtime)
// outlives an untouched one and a file with an unreadable header goes.
func (c *cache) fileExpired(path string, mtime, now time.Time) bool {
	if strings.HasSuffix(path, ".json") {
		return true
	}
	expires, ok := readHeaderExpiry(path)
	if !ok {
		return true
	}
	if mtime.After(expires) {
		expires = mtime
	}
	return !now.Before(expires.Add(staleGrace))
}

func readHeaderExpiry(path string) (time.Time, bool) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer func() { _ = f.Close() }()
	line, err := bufio.NewReaderSize(f, headerMaxBytes).ReadSlice('\n')
	if err != nil {
		return time.Time{}, false
	}
	var e entry
	if json.Unmarshal(line, &e) != nil {
		return time.Time{}, false
	}
	return e.Expires, true
}

// enforceCap removes the oldest surviving files by mtime until the
// directory fits maxDiskBytes (bounded by the per-pass delete budget).
func (c *cache) enforceCap(keep []os.FileInfo, deletes int) {
	var total int64
	for _, info := range keep {
		total += info.Size()
	}
	if total <= c.maxDiskBytes {
		return
	}
	sort.Slice(keep, func(i, j int) bool { return keep[i].ModTime().Before(keep[j].ModTime()) })
	for _, info := range keep {
		if total <= c.maxDiskBytes || deletes >= sweepMaxDeletes {
			return
		}
		if os.Remove(filepath.Join(c.dir, info.Name())) == nil {
			total -= info.Size()
			deletes++
		}
	}
}

// forget drops an entry from both tiers (F8): a cached body that no longer
// decodes must not be served until it expires.
func (c *cache) forget(rawURL string) {
	c.mu.Lock()
	if e, ok := c.mem[rawURL]; ok {
		c.bytes -= len(e.Body)
		delete(c.mem, rawURL)
	}
	c.mu.Unlock()
	if c.dir != "" {
		_ = os.Remove(c.path(rawURL))
	}
}

// readEntry parses a disk file: header line, then the raw body.
func readEntry(path string) (entry, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return entry{}, false
	}
	nl := bytes.IndexByte(raw, '\n')
	if nl < 0 {
		return entry{}, false
	}
	var e entry
	if json.Unmarshal(raw[:nl], &e) != nil {
		return entry{}, false
	}
	e.Body = raw[nl+1:]
	return e, true
}

// negative reports a remembered non-retryable failure.
func (c *cache) negative(rawURL string) (error, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n, ok := c.neg[rawURL]
	if !ok {
		return nil, false
	}
	if !c.now().Before(n.until) {
		delete(c.neg, rawURL)
		return nil, false
	}
	return n.err, true
}

// putNegative remembers a 4xx; at the cap the soonest-expiring entry is
// dropped so the map can never outgrow maxNegEntries (L1-F11).
func (c *cache) putNegative(rawURL string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.neg[rawURL]; !ok && len(c.neg) >= maxNegEntries {
		var victim string
		var soonest time.Time
		for k, n := range c.neg {
			if victim == "" || n.until.Before(soonest) {
				victim, soonest = k, n.until
			}
		}
		delete(c.neg, victim)
	}
	c.neg[rawURL] = negEntry{err: err, until: c.now().Add(NegativeTTL)}
}

// path is the disk file for a URL: SHA-256 of the full URL.
func (c *cache) path(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	return filepath.Join(c.dir, hex.EncodeToString(sum[:])+".cache")
}

// serverTTL reads the lifetime a server declares: Cache-Control max-age
// (unless no-store), else Expires relative to now. 0 = not cacheable.
func serverTTL(hdr http.Header) time.Duration {
	if hdr == nil {
		return 0
	}
	cc := strings.ToLower(hdr.Get("Cache-Control"))
	if strings.Contains(cc, "no-store") || strings.Contains(cc, "no-cache") {
		return 0
	}
	for _, part := range strings.Split(cc, ",") {
		part = strings.TrimSpace(part)
		if v, ok := strings.CutPrefix(part, "max-age="); ok {
			if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
				return time.Duration(secs) * time.Second
			}
		}
	}
	if exp, err := http.ParseTime(hdr.Get("Expires")); err == nil {
		if d := time.Until(exp); d > 0 {
			return d
		}
	}
	return 0
}

// Stats is a point-in-time view of the memory tier, the negative cache
// and the disk writer (Status modal / probes / the Q0 diagnostic dump).
type Stats struct {
	Entries    int
	Bytes      int
	Negative   int   // remembered 4xx URLs (≤ maxNegEntries)
	DiskWrites int64 // files the writer has written since launch (the Q1 gate's rate)
}

// stats reports the tiers' sizes.
func (c *cache) stats() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return Stats{Entries: len(c.mem), Bytes: c.bytes, Negative: len(c.neg), DiskWrites: c.diskWrites}
}

// flush waits until the writer has finished every item handed to it
// (tests): queued == handled, not merely an empty queue — the item in the
// writer's hands counts too (a loaded CI runner exposed the difference).
func (c *cache) flush() {
	if c.writes == nil {
		return
	}
	for i := 0; i < 5000 && c.handled.Load() < c.queued.Load(); i++ { // bounded wait (P10-02): ~5 s
		time.Sleep(time.Millisecond)
	}
}
