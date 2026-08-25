package main

import "testing"

func TestParseFoldsCountiesIntoTransmitters(t *testing.T) {
	js := `var ST = [];
ST[0] = "CA";
COUNTY[0] = "San Diego";
SAME[0] = "006073";
SITENAME[0] = "San Diego";
SITESTATE[0] = "CA";
FREQ[0] = "162.400";
CALLSIGN[0] = "KEC62";
LAT[0] = "32.84";
LON[0] = "-116.99";
PWR[0] = "1000";
STATUS[0] = "NORMAL";
ST[1] = "CA";
COUNTY[1] = "Orange";
SAME[1] = "006059";
SITENAME[1] = "San Diego";
SITESTATE[1] = "CA";
FREQ[1] = "162.400";
CALLSIGN[1] = "KEC62";
LAT[1] = "32.84";
LON[1] = "-116.99";
PWR[1] = "1000";
STATUS[1] = "NORMAL";
ST[2] = "CA";
COUNTY[2] = "Nowhere";
SAME[2] = "006999";
CALLSIGN[2] = "";
LAT[2] = "";
LON[2] = "";
`
	rows, err := Parse(js)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Callsign != "KEC62" || len(rows[0].SAME) != 2 || rows[0].SAME[0] != "006059" || rows[0].Lat != "32.84" {
		t.Fatalf("rows = %+v", rows)
	}
}
