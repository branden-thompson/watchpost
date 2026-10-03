package app

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
)

// THE RADIO PANEL'S CHOICES SURVIVE A RESTART (D-214): what the panel saves is
// written to the file and read back as the panel opens - a muted volume, zero,
// kept as zero and not the default.
func TestTheRadioPanelsChoicesSurviveARestart(t *testing.T) {
	for _, want := range []tty.RadioPrefs{{Volume: 30, Repeat: tty.RepeatWatchlist, Viz: true}, {Volume: 0, Repeat: tty.RepeatOne}} {
		withConfigFile(t)
		if err := saveRadioPrefs(want); err != nil {
			t.Fatal(err)
		}
		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}
		if got := radioPrefsFrom(cfg); got != want {
			t.Errorf("the panel opens on %+v; it saved %+v", got, want)
		}
	}
	if got := radioPrefsFrom(config.Config{}); got.Volume != 55 || got.Repeat != tty.RepeatOff || got.Viz {
		t.Errorf("with nothing kept the panel opens on %+v; want 55, Off, no visualizer", got)
	}
}

// THE RELAY'S PACING AND LANGUAGE SURVIVE A RESTART (D-214): Settings' choice
// is written to the file as well as handed to the deck, and the deck is given
// it at the next launch.
func TestTheRelaysPacingAndLanguageSurviveARestart(t *testing.T) {
	withConfigFile(t)
	lp := &livePipelines{}
	lp.setRelayDwell()(10 * time.Minute)
	lp.setRelayLang()("es")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	dwell, lang := relayPrefsFrom(cfg)
	if dwell != 10*time.Minute || lang != "es" {
		t.Errorf("the next launch's relay is %v, %q; it saved 10m, es", dwell, lang)
	}
}

// NO TEST WRITES THE DEVELOPER'S CONFIG: the package's TestMain points the
// config at a directory of the run's own, so a setter that keeps a preference
// never reaches the real file.
func TestNoTestWritesTheDevelopersConfig(t *testing.T) {
	path, err := config.Path()
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	if home != "" && strings.HasPrefix(path, home) {
		t.Errorf("the config is %s, under the home directory: a test that keeps a preference writes the real file", path)
	}
}

// THE KEPT RADIO REACHES THE DECK AND THE PANEL AT LAUNCH: what the file holds
// is the deck's relay pacing and language and the panel's choices.
func TestTheKeptRadioReachesTheDeckAndThePanelAtLaunch(t *testing.T) {
	vol := 20
	cfg := config.Config{Radio: config.Radio{Volume: &vol, Repeat: "one", Visualizer: true, RelayDwell: "10m", RelayLang: "es"}}
	model, err := tty.NewDashboard(tty.Config{})
	if err != nil {
		t.Fatal(err)
	}
	deck := &radioDeck{}
	got := keptRadio(deck, model, cfg).RadioPrefs()
	if got != (tty.RadioPrefs{Volume: 20, Repeat: tty.RepeatOne, Viz: true}) {
		t.Errorf("the panel opens on %+v; the file holds 20, One, visualizer on", got)
	}
	if deck.relayDwell != 10*time.Minute || deck.relayLang != "es" {
		t.Errorf("the deck has %v, %q; the file holds 10m, es", deck.relayDwell, deck.relayLang)
	}
}
