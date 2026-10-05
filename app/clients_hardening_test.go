package app

// clients_hardening_test.go — D-270, D-276: the station's shared data client
// and the CO-OPS client refuse to dial a private address and fetch over https
// only, as the map's clients do. The radio relay's directory, plain http by
// relay policy, has a client of its own: private addresses refused, http
// allowed.

import "testing"

func TestTheStationsClientsAreHardened(t *testing.T) {
	data := dataClientConfig(1)
	if !data.RefusePrivate || !data.HTTPSOnly {
		t.Errorf("the shared data client is %+v; want private addresses refused and https only (D-276)", data)
	}
	tides := coopsClientConfig()
	if !tides.RefusePrivate || !tides.HTTPSOnly {
		t.Errorf("the CO-OPS client is %+v; want private addresses refused and https only", tides)
	}
	dir := directoryClientConfig()
	if !dir.RefusePrivate || dir.HTTPSOnly {
		t.Errorf("the radio directory's client is %+v; want private addresses refused, http allowed (D-276)", dir)
	}
}
