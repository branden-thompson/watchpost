package httpx

import (
	"net/url"
	"strings"
	"testing"
)

// The two copies HostOf replaces, as 0.19.3 shipped them: app's map sources'
// and the temperature quota's. httpx's own string-slicing form was hostOf.
func appHostShipped(address string) string {
	u, err := url.Parse(address)
	if err != nil {
		return address
	}
	return u.Host
}

func quotaHostShipped(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	return u.Host
}

func sliceHostShipped(rawURL string) string {
	u := rawURL
	if i := strings.Index(u, "://"); i >= 0 {
		u = u[i+3:]
	}
	if i := strings.IndexAny(u, "/?"); i >= 0 {
		u = u[:i]
	}
	return u
}

// TestHostOfNamesTheHostEveryCopyNamed is 0.19.5 Q5.6 (DR-16): one HostOf for
// the Status window's map sources, the temperature quota and the failure
// columns. On every URL those callers pass (an absolute address, with a path,
// a query, a port or a tile template), it names what each copy named. The
// copies differed only where no caller goes: a URL with no host (app said "",
// the quota the URL), one that does not parse, and userinfo, which the slicing
// form kept (0.19.5 A-17).
func TestHostOfNamesTheHostEveryCopyNamed(t *testing.T) {
	for _, raw := range []string{
		"https://api.weather.gov/alerts/active?area=CA",
		"https://tile.openstreetmap.org/{z}/{x}/{y}.png",
		"https://mesonet.agron.iastate.edu:443/cgi-bin/wms/nexrad/n0q.cgi",
		"https://api.open-meteo.com/v1/forecast?latitude=33.2&longitude=-117.4",
		"https://earthquake.usgs.gov/fdsnws/event/1/query",
		"http://localhost:8080/debug",
	} {
		got := HostOf(raw)
		for name, want := range map[string]string{"app": appHostShipped(raw), "the quota": quotaHostShipped(raw), "the failure columns": sliceHostShipped(raw)} {
			if got != want {
				t.Errorf("HostOf(%q) = %q; %s named %q", raw, got, name, want)
			}
		}
	}
	if got := HostOf("not a url"); got != "not a url" {
		t.Errorf("an address with no host names %q; want the address itself", got)
	}
}
