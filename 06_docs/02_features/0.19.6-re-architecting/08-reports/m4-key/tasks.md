**T1.** Add a map Setting, *Forecast days*, that chooses which of each day's temperatures Forecast mode
draws: *Highs* (the default, what the map draws today) or *Lows*. Save it in `config.toml`. Show it as a
picker on the Settings window's Maps tab and as a picker in the map's Overlays window (`O`); changing it
in either place changes the same Setting, and the map shows the change at once. The `<` and `>` keys keep
flipping Forecast mode between highs and lows for the moment without changing the Setting; each time the
map opens, it starts from the Setting.

**T2.** Under the map, in Forecast mode with the UV layer on, the legend row reads `UV │ LOW … EXTREME`.
Listeners take "UV" for a unit. Make the row's head read `UV INDEX │` instead; nothing else on the row
changes.

**T3.** On the Broadcaster console, the UP NEXT card's DATA PULL line shows the date and the time of the
pull, for example `DATA PULL: 09/12/26  16:02:45  (2 MIN AGO)`. When the pull was made today, show only
the time and the age (`DATA PULL: 16:02:45  (2 MIN AGO)`); keep the date when it was made on an earlier
day. The DATA PULLED line in the card's detail window is not part of this change.

**T4.** When the station reads an alert aloud in a breaking takeover, alone or in a burst, it says when
the alert was declared: `at Four Fifty PM` for today, and `on November 13 at Eleven Fifty PM` for an
earlier day. Make it say `yesterday at Eleven Fifty PM` when the alert was declared the day before; keep
the date for anything older.
