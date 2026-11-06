package app

import (
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/report"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// TestTheOperatorsCardIsTheOperators is 0.19.5 Q2.5 (RN-6): the card an
// operator requests is minted here (D-40), as the operator's (FR-3.4), for
// the place and the reports they chose, admitted, and told to the Director at
// the position they typed.
func TestTheOperatorsCardIsTheOperators(t *testing.T) {
	var told []lineup.Event
	m := &mastercontrol{carry: func(ev lineup.Event) { told = append(told, ev) }}
	ref := snapshot.LocationRef{Label: "Fallbrook, CA", Lat: 33.37, Lon: -117.25}
	kinds := report.Set(0).Toggle(report.All()[0])
	m.RequestCard(ref, kinds, 4)
	if len(told) != 1 {
		t.Fatalf("the Director was told %d events; want one request", len(told))
	}
	req, ok := told[0].(lineup.Requested)
	if !ok {
		t.Fatalf("the Director was told %T; want lineup.Requested", told[0])
	}
	key := string(snapshot.Key(ref))
	c := req.Card
	if c.Origin != lineup.FromOperator {
		t.Errorf("the card's origin is %v; want the operator's: it loses its colour and label otherwise", c.Origin)
	}
	if c.Slot != lineup.LocationReport || c.ID != lineup.ReadID(key) || c.Subject != key || c.Headline != ref.Label {
		t.Errorf("the card is %+v; want a location report for %s", c, ref.Label)
	}
	if c.Reports != kinds {
		t.Errorf("the card carries reports %v; want the ones the operator ticked, %v", c.Reports, kinds)
	}
	if c.State != lineup.Admitted || req.To != 4 {
		t.Errorf("the card is %v at %d; want admitted at the typed position 4", c.State, req.To)
	}
}
