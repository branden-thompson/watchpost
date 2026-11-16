package app

import (
	"strings"
	"testing"
)

// todaysLayerOrder is the order Settings lists the map's layers in at 0.19.6
// B0, which is the order their init functions register them.
const todaysLayerOrder = "alert fire buoys tides quake radar rain temperature feels wind uv air waves"

// TestTheMapLayersKeepTheirOrder is 0.19.6 PB-4: the layers Settings lists are
// registered by the init functions of the map's files, so their order is the
// compiler's file order within one package, and package-dependency order once
// the registrants and the registry sit in different packages. A split that
// moved them would reorder the Settings rows without a line of code changing
// them. The order is pinned, so it changes only by an edit someone sees.
func TestTheMapLayersKeepTheirOrder(t *testing.T) {
	var keys []string
	for _, l := range windowLayers() {
		keys = append(keys, l.Key)
	}
	if got := strings.Join(keys, " "); got != todaysLayerOrder {
		t.Errorf("Settings lists the map's layers as %q; want %q, today's order", got, todaysLayerOrder)
	}
}
