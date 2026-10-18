package propanswers

// words.go — a target's bands and its typical day, in the words the
// Propagation mode says (FR-3.1, FR-1.15, D-127).

import (
	"strings"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// statusSaid are the bands' states in the order said: the open first.
var statusSaid = []struct {
	status ionomaps.Status
	words  string
}{
	{ionomaps.Open, "open on "},
	{ionomaps.Disturbed, "disturbed on "},
	{ionomaps.Absorbed, "absorbed on "},
	{ionomaps.Weak, "too weak on "},
	{ionomaps.AboveUpper, "above the upper limit on "},
}

// BandsSaid are a target's bands by state, the open before the closed:
// "open on 20m; above the upper limit on 17, 15, 12 and 10m".
func BandsSaid(bands []ionomaps.BandStatus) string {
	if len(bands) == 0 {
		return "no data"
	}
	var parts []string
	for _, s := range statusSaid {
		var nums []string
		for _, b := range bands {
			if b.Status == s.status {
				nums = append(nums, strings.TrimSuffix(b.Band.String(), "m"))
			}
		}
		if len(nums) > 0 {
			parts = append(parts, s.words+List(nums)+"m")
		}
	}
	if len(parts) == 0 {
		return "no data"
	}
	if !strings.HasPrefix(parts[0], "open on ") && !strings.HasPrefix(parts[0], "disturbed on ") {
		parts = append([]string{"nothing open"}, parts...) // the open and the disturbed are said first: neither leads, none is usable
	}
	return strings.Join(parts, "; ")
}

// List joins words as a sentence lists them: "a", "a and b", "a, b and c".
func List(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

// TypicalWords are the hours of the UTC day of hour each band is typically
// open in the local area (FR-1.15, AX-4): the chart's typical day, said.
func TypicalWords(bands []ionomaps.BandStatus, hour time.Time) string {
	if len(bands) == 0 {
		return "No typical day is known in the local area."
	}
	day := hour.UTC().Truncate(24 * time.Hour)
	end := day.Add(24 * time.Hour)
	var parts []string
	for _, b := range bands {
		var spans []string
		for _, sp := range b.OpenHours {
			from, to := sp.From, sp.To
			if from.Before(day) {
				from = day
			}
			if to.After(end) {
				to = end
			}
			if !from.Before(to) {
				continue
			}
			until := to.UTC().Format("15:04")
			if to.Equal(end) {
				until = "24:00"
			}
			spans = append(spans, from.UTC().Format("15:04")+" to "+until)
		}
		if len(spans) > 0 {
			parts = append(parts, b.Band.String()+" "+List(spans))
		}
	}
	if len(parts) == 0 {
		return "On a typical day no band is open in the local area."
	}
	return "On a typical day in the local area, in UTC, open: " + strings.Join(parts, "; ") + "."
}
