package propagation

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// TestALongValidatorIsNotKept is the REVIEW red team's RI-4: an ETag or a
// date past maxValidator is not kept, so the next ask asks whole and never
// echoes the service's bytes back; one of an ordinary size is kept.
func TestALongValidatorIsNotKept(t *testing.T) {
	for _, c := range []struct{ etag, want string }{{strings.Repeat("e", maxValidator+1), ""}, {`"abc123"`, `"abc123"`}} {
		rec := &recorder{answer: func(string) (httpx.Answer, error) {
			return httpx.Answer{Status: 200, Header: http.Header{"Etag": {c.etag}}}, nil
		}}
		f := &fetcher{get: rec, hosts: map[string]bool{"services.swpc.noaa.gov": true}, note: func(string) {}, seen: map[string]bool{}}
		r, err := f.Fetch(context.Background(), "https://services.swpc.noaa.gov/text/daily-solar-indices.txt", ionomaps.Validators{})
		if err != nil || r.Validators.ETag != c.want {
			t.Errorf("an ETag of %d bytes: kept %q (%v); want %q", len(c.etag), r.Validators.ETag, err, c.want)
		}
	}
}

// TestAStateFilesErrorNamesTheFileAlone is the REVIEW red team's RI-7: a
// failure to read or keep the throttle state is noted with the file's name,
// never its path, which holds the listener's home.
func TestAStateFilesErrorNamesTheFileAlone(t *testing.T) {
	dir := t.TempDir()
	_, err := os.ReadFile(dir + "/not-a-file/state.json")
	if got := fileOnly(err); strings.Contains(got, dir) || !strings.Contains(got, "state.json") {
		t.Errorf("a read's error says %q; want the file's name alone", got)
	}
	if got := fileOnly(&os.LinkError{Op: "rename", Old: dir + "/a.tmp", New: dir + "/state.json", Err: os.ErrPermission}); strings.Contains(got, dir) {
		t.Errorf("a rename's error says %q", got)
	}
}
