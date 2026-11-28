package run

// shared.go — what one run of watchpost shares across the root, the deck
// and the map data: how it names itself, where it caches, how it saves a
// preference, the locked copy and the merged request counts. every.go holds
// the one clock-driven loop beside them (0.19.6 design AP-5).

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// UserAgent identifies watchpost to providers (0.9.0 AI-1: NWS requires
// contact info; a reachable project URL satisfies it). The tree is public, so
// no personal address rides here.
//
// NO VERSION. A version in the header is a fact about this machine that no
// provider asks for, and one more thing to keep in step with the build. The
// project URL is the contact NWS wants.
const UserAgent = "watchpost (+https://github.com/branden-thompson/watchpost)"

// SavePreference is the no-error convenience over config.Mutate, for the
// preference writes that cannot fail. It is an ADAPTER over config.Mutate, not
// a second write path (0.15.0 FR-1.1).
func SavePreference(edit func(cfg *config.Config)) error {
	return config.Mutate(func(cfg *config.Config) error {
		edit(cfg)
		return nil
	})
}

// CacheDir is the on-disk tier of the HTTP cache (0.9.0 UAT 71): the OS cache
// directory ($XDG_CACHE_HOME or ~/Library/Caches on macOS), never the config
// dir — it holds only public weather products and is safe to delete. ""
// (memory only) when the OS gives us no cache directory.
func CacheDir() string { return UserCacheSubdir("http") }

// UserCacheSubdir is <OS cache dir>/watchpost/<name> — the one builder for
// the HTTP cache and the voice install; "" when the OS has
// no cache directory (the callers degrade: memory only, no voice).
func UserCacheSubdir(name string) string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "watchpost", name)
}

// LockedCopy is a copy of a slice the deck's lock guards, read under it.
func LockedCopy[T any](mu *sync.Mutex, src *[]T) []T {
	mu.Lock()
	defer mu.Unlock()
	return append([]T(nil), *src...)
}

// RequestStatsOf is the asks per host of every client there is, merged.
func RequestStatsOf(clients []*httpx.Client) httpx.RequestStats {
	all := make([]httpx.RequestStats, 0, len(clients))
	for _, c := range clients {
		if c != nil {
			all = append(all, c.RequestStats())
		}
	}
	return httpx.MergeRequestStats(all...)
}
