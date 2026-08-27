package nws

// forecast.go — the daily and hourly forecasts and the gridpoint max/min fill. Split from provider.go by the quality pass (Q2, pure move).

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/tz"
)

// --- forecast ---

type period struct {
	StartTime                  time.Time `json:"startTime"`
	IsDaytime                  bool      `json:"isDaytime"`
	Temperature                float64   `json:"temperature"`
	TemperatureUnit            string    `json:"temperatureUnit"`
	ProbabilityOfPrecipitation quantity  `json:"probabilityOfPrecipitation"`
	WindSpeed                  string    `json:"windSpeed"`
	ShortForecast              string    `json:"shortForecast"`
}

// periodsDoc is the /forecast and /forecast/hourly payload shape.
type periodsDoc struct {
	Properties struct {
		Periods []period `json:"periods"`
	} `json:"properties"`
}

// fetchForecast is the daily forecast (KindForecast): 12-hour periods
// folded into calendar days, holes filled from the gridpoint.
func (p *Provider) fetchForecast(ctx context.Context, ref snapshot.LocationRef) (snapshot.PartialData, error) {
	g, err := p.resolve(ctx, ref)
	if err != nil {
		return snapshot.PartialData{}, err
	}
	var daily periodsDoc
	if _, err := p.client.GetJSON(ctx, g.forecastURL, &daily); err != nil {
		return snapshot.PartialData{}, fmt.Errorf("forecast for %s: %w", ref.Label, err)
	}
	pd := snapshot.PartialData{Daily: foldDaily(daily.Properties.Periods)}
	p.fillDailyFromGrid(ctx, g, pd.Daily)
	return pd, nil
}

// fetchHourly is the hourly forecast (KindForecastHourly, UAT 72 — its own
// tier: 162 KB per location, so the RECENT list hydrates it on demand).
func (p *Provider) fetchHourly(ctx context.Context, ref snapshot.LocationRef) (snapshot.PartialData, error) {
	g, err := p.resolve(ctx, ref)
	if err != nil {
		return snapshot.PartialData{}, err
	}
	var hourly periodsDoc
	if _, err := p.client.GetJSON(ctx, g.hourlyURL, &hourly); err != nil {
		return snapshot.PartialData{}, fmt.Errorf("hourly forecast for %s: %w", ref.Label, err)
	}
	pd := snapshot.PartialData{}
	for _, h := range hourly.Properties.Periods {
		t := tempC(h.Temperature, h.TemperatureUnit)
		pd.Hourly = append(pd.Hourly, snapshot.Hourly{
			Time:       h.StartTime,
			Temp:       &t,
			PrecipProb: h.ProbabilityOfPrecipitation.Value,
			Wind:       windFromText(h.WindSpeed),
			Condition:  conditionCode(h.ShortForecast),
		})
	}
	return pd, nil
}

// fillDailyFromGrid fills a day's missing HIGH/LOW from the raw gridpoint
// maxTemperature/minTemperature series (B3 UAT 71): /forecast drops a
// day's daytime period once local evening starts, so TODAY's HIGH read
// "n/a" east of wherever 6 PM had passed. The gridpoint keeps the value
// all day, and nws-marine already fetches it — the client cache makes this
// one download per grid per cycle. A nicety: any failure leaves the hole.
func (p *Provider) fillDailyFromGrid(ctx context.Context, g *gridInfo, daily []snapshot.Daily) {
	if g.gridURL == "" || !hasTempHole(daily) {
		return
	}
	var grid struct {
		Properties struct {
			Max gridSeries `json:"maxTemperature"`
			Min gridSeries `json:"minTemperature"`
		} `json:"properties"`
	}
	if _, err := p.client.GetJSON(ctx, g.gridURL, &grid); err != nil {
		return
	}
	tz, err := tz.Location(g.timeZone)
	if err != nil {
		tz = time.UTC
	}
	for i := range daily {
		d := &daily[i]
		if d.TempMax == nil {
			if v, ok := grid.Properties.Max.extremeOn(d.Date, tz, true); ok {
				d.TempMax, d.FillFrom = &v, fillNote(d.FillFrom, "temp_max")
			}
		}
		if d.TempMin == nil {
			if v, ok := grid.Properties.Min.extremeOn(d.Date, tz, false); ok {
				d.TempMin, d.FillFrom = &v, fillNote(d.FillFrom, "temp_min")
			}
		}
	}
}

func hasTempHole(daily []snapshot.Daily) bool {
	for _, d := range daily {
		if d.TempMax == nil || d.TempMin == nil {
			return true
		}
	}
	return false
}

func fillNote(m map[string]string, field string) map[string]string {
	if m == nil {
		m = map[string]string{}
	}
	m[field] = "nws:gridpoint"
	return m
}

// extremeOn is the max (or min) of a series' values whose validity starts
// on date in tz, in °C rounded to a tenth; false when the date has none.
func (s gridSeries) extremeOn(date string, tz *time.Location, wantMax bool) (float64, bool) {
	best, found := 0.0, false
	for _, v := range s.Values {
		if v.Value == nil {
			continue
		}
		start, err := time.Parse(time.RFC3339, strings.SplitN(v.ValidTime, "/", 2)[0])
		if err != nil || start.In(tz).Format("2006-01-02") != date {
			continue
		}
		val := *v.Value
		if s.UOM == "wmoUnit:degF" {
			val = (val - 32) * 5 / 9
		}
		if !found || (wantMax && val > best) || (!wantMax && val < best) {
			best, found = val, true
		}
	}
	return roundTenth(best), found
}

// foldDaily folds NWS 12-hour day/night periods into calendar days.
func foldDaily(periods []period) []snapshot.Daily {
	byDate := map[string]*snapshot.Daily{}
	var order []string
	for _, per := range periods {
		date := per.StartTime.Format("2006-01-02")
		d, ok := byDate[date]
		if !ok {
			d = &snapshot.Daily{Date: date}
			byDate[date] = d
			order = append(order, date)
		}
		t := tempC(per.Temperature, per.TemperatureUnit)
		if per.IsDaytime {
			d.TempMax = &t
			d.Condition = conditionCode(per.ShortForecast)
		} else {
			d.TempMin = &t
			if d.Condition == "" {
				d.Condition = conditionCode(per.ShortForecast)
			}
		}
		if v := per.ProbabilityOfPrecipitation.Value; v != nil {
			if d.PrecipProb == nil || *v > *d.PrecipProb {
				d.PrecipProb = v
			}
		}
	}
	sort.Strings(order)
	out := make([]snapshot.Daily, 0, len(order))
	for _, date := range order {
		out = append(out, *byDate[date])
	}
	return out
}

func tempC(v float64, unit string) float64 {
	if unit == "F" {
		return roundTenth((v - 32) * 5 / 9)
	}
	return roundTenth(v)
}

func roundTenth(v float64) float64 {
	if v < 0 {
		return float64(int(v*10-0.5)) / 10
	}
	return float64(int(v*10+0.5)) / 10
}
