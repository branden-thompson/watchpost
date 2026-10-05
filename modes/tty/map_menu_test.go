package tty

// map_menu_test.go — 0.18.0 U2-39, D-141 to D-146: the HUM LEAD's
// consolidated MAP DETAILS / OVERLAYS menu.

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// menuMap is the map with every group's layers registered, in Forecast mode.
func menuMap(t *testing.T) Dashboard {
	t.Helper()
	var asks []MapAsk
	d := openTempMap(t, false, &asks)
	d.cfg.MapLayers = append(d.cfg.MapLayers, MapLayer{Key: FireLayer, Label: "Fire", On: true}, MapLayer{Key: QuakeLayer, Label: "Earthquakes", On: true},
		MapLayer{Key: BuoyLayer, Label: "Buoys"}, MapLayer{Key: TideLayer, Label: "Tides"})
	return d
}

// at puts the menu's cursor on a row by its key.
func at(t *testing.T, d Dashboard, key string) Dashboard {
	t.Helper()
	for i, r := range d.overlayRows() {
		if r.key == key {
			d.mapPane.menuAt, d.mapPane.menuOn = i, true
			return d
		}
	}
	t.Fatalf("no row %q", key)
	return d
}

// menuKey is a key to the open menu.
func menuKey(t *testing.T, d Dashboard, key string) Dashboard {
	t.Helper()
	nd, ok := d.handleOverlaysKey(key)
	if !ok {
		t.Fatalf("the open menu did not own %q", key)
	}
	return nd
}

// TestTheMenuIsTheHUMLEADsLayout is U2-39 with D-141: the tints as radio
// rows; Data Points, Hazards and Alert Areas, each its switch over its
// boxes; the preset over the detail switches; Rain & snow in Forecast mode
// alone; the warning at the top past the thresholds, and not below them.
func TestTheMenuIsTheHUMLEADsLayout(t *testing.T) {
	d := menuMap(t)
	var got []string
	for _, r := range d.overlayRows() {
		got = append(got, r.key)
	}
	want := "temperature uv air group:points wind waves buoys tides rain group:hazards quake fire alert alert-emergency alert-warnings alert-watches alert-advisories alert-statements alert-marine level borders water rivers names roads rail parks"
	if strings.Join(got, " ") != want {
		t.Errorf("the rows are\n%s\nwant\n%s", strings.Join(got, " "), want)
	}
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: 'R', Text: "R"})
	if menuHas(settleRadar(t, m.(Dashboard), cmd), RainLayer) {
		t.Error("Radar mode lists Rain & snow: the radar is its rain (D-117)")
	}
	quiet := stripANSITest(strings.Join(d.overlaysBox(), "\n"))
	d.mapCost = MapCost{Bytes: 3_000_000, Requests: 100}
	loud := stripANSITest(strings.Join(d.overlaysBox(), "\n"))
	if strings.Contains(quiet, "performance") || !strings.Contains(loud, "! You may experience performance") {
		t.Errorf("the warning at the menu's top (D-146):\n%s", loud)
	}
	for _, w := range []string{"● Temperature", "[←] Actual", "○ UV Index", "Data Points", "[←] Enabled", "Hazards", "Quakes", "[←] All", "Alert Areas", "Preset:", "[←] Standard"} {
		if !strings.Contains(loud, w) {
			t.Errorf("the menu lacks %q:\n%s", w, loud)
		}
	}
}

// TestSpaceOnTheChosenTintClearsIt is D-142: space chooses a tint, turning
// the others off; space on the chosen one clears it, no tint at all.
func TestSpaceOnTheChosenTintClearsIt(t *testing.T) {
	d := at(t, menuMap(t), UVLayer)
	d = menuKey(t, d, "space")
	if !d.layerOn(UVLayer) || d.layerOn(TemperatureLayer) {
		t.Fatalf("space on UV: UV %v, temperature %v", d.layerOn(UVLayer), d.layerOn(TemperatureLayer))
	}
	d = menuKey(t, d, "space")
	for _, k := range oneTint {
		if d.ticked(k) {
			t.Errorf("space on the chosen UV left %s on; want no tint", k)
		}
	}
}

// TestADisabledGroupHidesAndRemembers is D-143: a group switched off draws
// none of its layers, their ticks kept and dimmed; on again, as they were.
func TestADisabledGroupHidesAndRemembers(t *testing.T) {
	d := at(t, menuMap(t), WindLayer)
	d = menuKey(t, d, "space") // Wind ticked
	d = at(t, d, groupPoints)
	d = menuKey(t, d, "right") // Data Points: Disabled
	if d.layerOn(WindLayer) || !d.ticked(WindLayer) || d.mapAsk().Tides {
		t.Fatalf("Data Points off: wind drawn %v, ticked %v", d.layerOn(WindLayer), d.ticked(WindLayer))
	}
	if box := stripANSITest(strings.Join(d.overlaysBox(), "\n")); !strings.Contains(box, "[←] Disabled") || !strings.Contains(box, "[✔] Wind") {
		t.Errorf("the disabled group:\n%s", box)
	}
	d = menuKey(t, d, "space") // Enabled again
	if !d.layerOn(WindLayer) {
		t.Error("Data Points on again: wind is not drawn as it was ticked")
	}
	d = at(t, d, AlertLayer)
	d = menuKey(t, d, "space")
	if d.layerOn(AlertLayer) {
		t.Error("Alert Areas' switch is not the alert layer's")
	}
}

// TestTheFireRowChoosesWhatIsDrawn is D-145: ←→ on Fire steps All, Named,
// Hotspots, and the ask carries the choice; the arrows on a row without a
// choice move nothing, the menu owning them.
func TestTheFireRowChoosesWhatIsDrawn(t *testing.T) {
	d := at(t, menuMap(t), FireLayer)
	for _, want := range []string{FireNamed, FireHotspots, FireAll} {
		d = menuKey(t, d, "right")
		if d.fireMode() != want || d.mapAsk().FireMode != want {
			t.Errorf("→ on Fire: %q, asked %q; want %q", d.fireMode(), d.mapAsk().FireMode, want)
		}
	}
	d = menuKey(t, d, "left")
	if d.fireMode() != FireHotspots {
		t.Errorf("← from All is %q; want Hotspots", d.fireMode())
	}
	d = at(t, d, WindLayer)
	before := d.mapLayerChoice
	if d = menuKey(t, d, "left"); d.mapLayerChoice != before {
		t.Error("← on a box changed a choice")
	}
}

// TestSettingsKeepsOneTintToo is D-119, D-137 and D-139 in the Settings
// window's Layers row: switching UV on there switches temperature off, as the
// menu does.
func TestSettingsKeepsOneTintToo(t *testing.T) {
	d := menuMap(t)
	for i, l := range d.cfg.MapLayers {
		if l.Key == UVLayer {
			d.setup.layerAt = i
		}
	}
	d = d.toggleLayer()
	if !d.ticked(UVLayer) || d.ticked(TemperatureLayer) || d.ticked(FeelsLayer) {
		t.Errorf("Settings: UV %v, temperature %v; want UV alone", d.ticked(UVLayer), d.ticked(TemperatureLayer))
	}
}

// TestTheMenusPickersAreTheAppsPickers is D-147: a choice row is Settings'
// picker, [←] value [→]; the chip pressed blinks on that row alone, as
// Settings' do, and the tick after its window ends the blink.
func TestTheMenusPickersAreTheAppsPickers(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(rendering.ResetColorEnabledForTest)
	d := at(t, menuMap(t), FireLayer)
	d = menuKey(t, d, "right")
	o := d.opts()
	lit := o.KeyCapInverted("→")
	fire, box := "", d.overlaysBox()
	for _, l := range box {
		if strings.Contains(stripANSITest(l), "Fire") {
			fire = l
		}
	}
	if !strings.Contains(fire, lit) || strings.Count(strings.Join(box, "\n"), lit) != 1 {
		t.Errorf("→ on Fire: its row is %q; want its → chip lit, and no other", fire)
	}
	if plain := stripANSITest(fire); !strings.Contains(plain, "←") || !strings.Contains(plain, "Named") || !strings.Contains(plain, "→") {
		t.Errorf("the Fire row is %q; want the picker, Named", stripANSITest(fire))
	}
	d.mapPane.menuFlashEnd = d.mapPane.menuFlashEnd.Add(-time.Hour)
	if got := d.applyTick(); got.mapPane.menuFlash != flashNone || strings.Contains(strings.Join(got.overlaysBox(), ""), lit) {
		t.Error("the tick after the blink's window did not end it")
	}
	d = at(t, d, WindLayer)
	if d = menuKey(t, d, "right"); d.mapPane.menuFlash != flashNone && d.mapPane.menuFlashAt == d.mapPane.menuAt {
		t.Error("→ on a box with no picker blinked a chip")
	}
}

// TestTheMenuBlinkKeepsATickUntilItEnds is W14's C-3: the Overlays menu's
// picker blink is cleared by the tick (applyTick), so a tick must be armed
// while it shows - as Settings' picker blink keeps one. Nothing else here
// needs a tick, which the control asserts, so the blink alone must.
func TestTheMenuBlinkKeepsATickUntilItEnds(t *testing.T) {
	d := mapDash(t, Config{})
	d, _ = pressKey(d, "g")
	d.ticker = nil
	if d.tickNeeded() {
		t.Fatal("control: something else keeps a tick armed here, so this test proves nothing")
	}
	d.mapPane.menuFlash, d.mapPane.menuFlashEnd = flashRight, time.Now().Add(time.Second)
	if !d.tickNeeded() {
		t.Error("the menu's blink shows with no tick armed: nothing will clear it")
	}
}

// TestEveryBoxIsDrawnOnce is the menu's boxes two to a line: a box drawn
// beside the one before it is not drawn again on a line of its own.
func TestEveryBoxIsDrawnOnce(t *testing.T) {
	d := menuMap(t)
	plain := stripANSITest(strings.Join(d.overlaysBox(), "\n"))
	boxes := 0
	for _, r := range d.overlayRows() {
		if r.kind == menuRadio || r.kind == menuGroup || r.kind == menuFire || r.kind == menuPreset {
			continue
		}
		boxes++
		on := d.detailOn(r.key)
		if r.weather {
			on = d.ticked(r.key)
		}
		cell := regexp.MustCompile(regexp.QuoteMeta(stripANSITest(checkMark(d.opts(), on))+" "+r.label) + `( |$)`)
		if n := len(cell.FindAllStringIndex(plain, -1)); n != 1 {
			t.Errorf("the box %q is drawn %d times:\n%s", r.label, n, plain)
		}
	}
	if boxes < 4 {
		t.Fatalf("the fixture has %d boxes; this test measures nothing", boxes)
	}
}
