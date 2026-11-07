package tty

// dashboard_choices.go — the Settings the model is built with, in three
// groups (0.19.5 D-14, A-16). Each group is a struct embedded in Dashboard, so
// its fields read as the model's own, and each is built from the Config by one
// keyed literal, one field per line. A new Setting lands in its group's struct
// and its builder.

import (
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/render"
)

// mapChoices are the map's Settings (0.18.0 onward). The Maps tab's scale the
// map opens at, its nearby distance and its layer and detail choices as
// comparable words (0.18.0 batch 9) are among them.
type mapChoices struct {
	mapsOff         bool        // 0.18.0: the Setting; g says so and builds nothing (W1.8)
	mapDesc         mapDescMode // 0.18.0: the description with the picture, instead of it, or off (W1.10)
	mapScale        mapScaleMode
	mapNearbyKm     int
	mapLayerChoice  string
	mapDetailChoice string         // the map's detail choices as one comparable word (D-65)
	mapDetailLevel  tuimaps.Detail // how much of the basemap is drawn (D-67, go-tuiMaps D-82)
	mapRadarIEM     bool           // IEM for the lower 48's radar, else MRMS (D-83)
	mapTempNDFD     bool           // NDFD for the map's temperature, both modes, else Open-Meteo (D-93, D-190)
	mapRainFull     bool           // Open-Meteo's full density for the rain past NDFD's reach (D-192)
	mapUVCities     int            // how many cities UV asks EPA for: 8, 24 or 48 (D-202)
	mapPropRadiusKm int            // the Propagation mode's near-vertical radius: 200, 400 or 600 km (D-100)
	mapPropNoise    ionomaps.Noise // the reference circuit's noise environment (D-187)
	mapPropRefresh  propRefresh    // how often an open mode asks again (D-120)
	mapPropCorrect  bool           // GloTEC corrected by the typical offset while no readings are held (D-105)
	mapPropNoGIRO   bool           // GIRO not asked: NOAA's and the climatology's field alone (D-214)
	mapDepth        mapDepth       // the map's colour depth (D-207)
	mapPlayback     time.Duration  // how long each frame of the map's playback shows (D-361)
	mapRadarAhead   int            // the radar loop's hours ahead: 1, 3, 6 or 12 (D-114)
	mapQuakeFeed    string         // the quakes drawn: USGS's feed by its name (D-122)
}

// mapChoicesOf reads the map's Settings from the Config; a word the file does
// not offer is the Setting's default.
func mapChoicesOf(cfg Config) mapChoices {
	return mapChoices{
		mapsOff:         cfg.Maps == "off",
		mapDesc:         mapDescByKey(cfg.MapDescription),
		mapScale:        mapScaleByKey(cfg.MapScale),
		mapNearbyKm:     mapNearbyByKm(cfg.MapNearbyKm),
		mapLayerChoice:  layerChoiceKey(cfg.MapLayerChoice),
		mapDetailChoice: layerChoiceKey(cfg.MapDetailChoice),
		mapDetailLevel:  detailLevelByKey(cfg.MapDetailLevel),
		mapRadarIEM:     cfg.MapRadarSource == "iem",
		mapTempNDFD:     cfg.MapTempSource != tempSourceOpenMeteo,
		mapRainFull:     cfg.MapRainDetail == rainDetailFull,
		mapUVCities:     UVCitiesByCount(cfg.MapUVCities),
		mapPropRadiusKm: propRadiusByKm(cfg.MapPropRadius),
		mapPropNoise:    propNoiseByKey(cfg.MapPropNoise),
		mapPropRefresh:  propRefreshByKey(cfg.MapPropRefresh),
		mapPropCorrect:  cfg.MapPropNoReadings == noReadingsCorrected,
		mapPropNoGIRO:   cfg.MapPropStations == noStationsKey,
		mapDepth:        mapDepthByKey(cfg.MapColourDepth),
		mapPlayback:     playbackByKey(cfg.MapPlaybackMs),
		mapRadarAhead:   radarAheadByHours(cfg.MapRadarAhead),
		mapQuakeFeed:    quakeFeedByKey(cfg.MapQuakeFeed),
	}
}

// uiChoices are the display Settings both surfaces share.
type uiChoices struct {
	units        render.Units
	clockFmt     render.Clock // how times of day are written (render/clock.go); `clock()` is the wall clock
	reduceMotion bool         // nothing moves on its own clock (D-360)
}

// uiChoicesOf reads the display Settings from the Config.
func uiChoicesOf(cfg Config) uiChoices {
	return uiChoices{
		units:        render.UnitsByKey(cfg.Units),
		clockFmt:     render.ClockByKey(cfg.Clock),
		reduceMotion: cfg.ReduceMotion,
	}
}

// radioChoices are the radio's Settings: the correspondent, and the player's
// controls, which the radio panel sets.
type radioChoices struct {
	radioVoice  string     // the chosen correspondent (chip label)
	radioVolume int        // 0-100 (D-19); [+]/[-] step 5, bar cells step at the 10s (UAT 41)
	radioRepeat RepeatMode // [r] Off | One | Watchlist (UAT 93)
	radioMode   RadioMode  // [m] Synth | Nearest Relay (UAT 97)
	radioViz    bool
}

// radioChoicesOf reads the radio's Settings from the Config; the player's
// controls open at their defaults, the volume at 55.
func radioChoicesOf(cfg Config) radioChoices {
	return radioChoices{
		radioVoice:  cfg.Voice,
		radioVolume: 55,
	}
}
