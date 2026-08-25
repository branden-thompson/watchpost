package httpx

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// cache is the URL-keyed response store: a memory tier that every Client
// has, and an optional disk tier (CacheDir) holding the same entries so a
// relaunch is warm. Disk files are a one-line JSON header (redacted URL,
// expiry) followed by the raw body — inspectable, and read back without
// decoding (UAT 73: the earlier base64 format cost a third more bytes and
// a full copy per read). Only public weather data ever lands here.
type cache struct {
	dir    string
	writes chan entry    // disk writes happen on one goroutine, off the request path (UAT 73)
	reads  chan struct{} // bounds concurrent disk reads (UAT 74): a 200-goroutine warm launch once spawned ~90 OS threads on short file syscalls

	mu    sync.Mutex
	mem   map[string]*entry
	bytes int
	tick  uint64 // LRU clock
	neg   map[string]negEntry
}

// entry is one cached response.
type entry struct {
	URL     string    `json:"url"` // redacted, for humans reading the file
	Expires time.Time `json:"expires"`
	Body    []byte    `json:"-"`
	key     string
	used    uint64
}

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

func newCache(dir string) *cache {
	c := &cache{dir: dir, mem: map[string]*entry{}, neg: map[string]negEntry{}, reads: make(chan struct{}, maxDiskRead)}
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
// it fits the memory budget).
func (c *cache) get(rawURL string) ([]byte, bool) {
	now := time.Now()
	c.mu.Lock()
	if e, ok := c.mem[rawURL]; ok && now.Before(e.Expires) {
		c.tick++
		e.used = c.tick
		c.mu.Unlock()
		return e.Body, true
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

// put stores a body in memory (budgeted) and queues it for disk.
func (c *cache) put(rawURL string, body []byte, expires time.Time) {
	e := entry{URL: RedactURL(rawURL), Expires: expires, Body: body}
	c.remember(rawURL, e)
	if c.writes == nil {
		return
	}
	e.key = rawURL
	select {
	case c.writes <- e:
	default: // queue full: this entry just misses the warm relaunch
	}
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

// evictLocked drops expired entries, then least-recently-used ones, until
// the tier is within budget (caller holds mu).
func (c *cache) evictLocked() {
	now := time.Now()
	for k, e := range c.mem {
		if !now.Before(e.Expires) {
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

// writer persists queued entries one at a time (atomic rename).
func (c *cache) writer() {
	for e := range c.writes {
		hdr, err := json.Marshal(e)
		if err != nil {
			continue
		}
		path := c.path(e.key)
		// A unique temp name (red-team 0.9.0 F8): `report` and the dashboard
		// share this directory and may write the same entry at once.
		f, err := os.CreateTemp(c.dir, filepath.Base(path)+".*.tmp")
		if err != nil {
			continue
		}
		tmp := f.Name()
		_, werr := f.Write(append(append(hdr, '\n'), e.Body...))
		if cerr := f.Close(); werr == nil && cerr == nil && os.Chmod(tmp, 0o600) == nil {
			_ = os.Rename(tmp, path)
		} else {
			_ = os.Remove(tmp)
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
	if !time.Now().Before(n.until) {
		delete(c.neg, rawURL)
		return nil, false
	}
	return n.err, true
}

func (c *cache) putNegative(rawURL string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.neg[rawURL] = negEntry{err: err, until: time.Now().Add(NegativeTTL)}
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

// Stats is a point-in-time view of the memory tier (Status modal / probes).
type Stats struct {
	Entries int
	Bytes   int
}

// stats reports the memory tier's size.
func (c *cache) stats() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return Stats{Entries: len(c.mem), Bytes: c.bytes}
}

// flush waits for queued disk writes (tests).
func (c *cache) flush() {
	if c.writes == nil {
		return
	}
	for i := 0; i < 1000 && len(c.writes) > 0; i++ { // bounded wait (P10-02): ~1 s
		time.Sleep(time.Millisecond)
	}
	time.Sleep(5 * time.Millisecond)
}
