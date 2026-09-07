# Data shape of record — roles, voices, tones (multi-voice-support, 0.14.0)

The one place the role registry, the config keys, the resolution walk and the fallback matrix are defined
(red-team A-1/D-1: two shapes had been on the record). PLAN builds from this document; the earlier block in
`voice-architecture.md` is superseded.

## 1. The role registry (FR-1)

| Node | Key | Parent | Speaks |
|---|---|---|---|
| All reads | `voice` (the 0.13.0 root; unchanged) | — | everything not assigned below |
| Alerts & Notification reads | `alerts` | root | the two alert roles |
| ↳ Breaking takeover | `breaking` | `alerts` | the ticker's takeover lines and its tone's rate |
| ↳ Severe-event read | `severe_read` | `alerts` | the `[space]` read from the severe window |
| Standard reports | `standard` | root | the five report roles |
| ↳ Location weather | `weather` | `standard` | lead notice/span, conditions, alerts, the NWS products |
| ↳ Location maritime | `maritime` | `standard` | the maritime report (FR-6) |
| ↳ Location fire & hotspots | `fire` | `standard` | the fire report |
| ↳ Location seismic | `seismic` | `standard` | the seismic report |
| ↳ Station | `station` | `standard` | the station lead ("This is the <report> for Watchpost Radio.") and the sign-off (MVS-D-13) |

Nine assignable nodes below the root. The field list of a Go struct **is** the registry — adding a role is one
field plus one `Segment.Role` constant; a typo'd key in a hand-edited file is ignored (the config contract,
`platform/config/config.go:136-137`) and listed once in `[S]` (NFR-5), never minted as a role.

## 2. The config keys (FR-8, NFR-5) — additive, zero value = inherit

```toml
voice = "Samantha"                 # unchanged — the root; a single string for compatibility (AM-11)

[radio]
mode = "synth"                     # unchanged

[radio.voices.alerts]              # one typed pair per role: the macOS `say -v ?` name and the Piper catalogue KEY
macos = "Rishi"                    # "" = inherit on this OS
piper = "en_US-ryan-medium"        # keys, never display names

[radio.voices.breaking]            # the other eight roles look the same; an absent table = inherit
macos = ""
piper = ""
# standard · severe_read · weather · maritime · fire · seismic · station

cast = ""                          # MVS-D-25: "" = Single Voice (the root reads everything; the pairs are kept but unread) | "cast"

[radio.tones]                      # MVS-D-26: a per-class MUTE; every class always sounds its ratified preset
mode  = ""                         # "" = All Tones On | "mute" = the classes in `muted` are silent ([M] toggles this)
muted = ["watch", "advisory"]      # of: disaster · warning · watch · advisory · statement · storm
```

`Voices` is a struct of nine `RoleVoice{MacOS, Piper string}` fields (red-team B-6: a struct of structs — a
typo'd role key is ignored, never minted; `omitempty` omits an empty pair so an untouched file stays
byte-identical); each OS **reads and writes only its own field of the pair**, and the other survives a save,
so a file synced between a Mac and a Linux box keeps both assignments. 0.14.0's save also preserves keys it does
not know (a shadow map merged on write, red-team B-1) so a *future* binary's keys survive too. `Tones{Mode string; Muted []string}` — `Mode` a closed set (`""` | `"mute"`), `Muted` a list over the six
class keys (`disaster` · `warning` · `watch` · `advisory` · `statement` · `storm`). `[M]` flips `Mode` and
persists it; the set survives the flip (the mock's "All Tones On (Default) / Mute:"). **Compatibility:** a
0.13.0 file's `ticker_muted = true` with no `[radio.tones]` loads as `Mode = "mute", Muted = all six`; 0.14.0
keeps writing `ticker_muted` (= `Mode == "mute"`) so a 0.13.0 binary still mutes the ticker. `Radio.Validate()`
runs at load beside `Fire.Validate` (`config.go:197`): a wrong type names the key; a mode or class outside its
set names the key.

Compatibility: a 0.13.0 binary ignores every new key and drops the tables on its next save (NFR-5); a 0.14.0
binary reading a 0.13.0 file sees empty tables → everything inherits the root → FR-2.

## 3. Resolution (FR-7) — one owner, `resolveVoice(role) (synth.Voice, Resolution)`

Walk `[role, group, root, platform default]`; at each link read **this host's namespace** (macOS table on
darwin, Piper table elsewhere); the first link that **exists on this host** wins:

- macOS: the name is on the host's list — the curated `macVoices()` list until `say -v ?` has answered, the
  intersection after (a closed allowlist at every moment; D-R2-2 deleted the "trust until it lands" window)
  **or** is the `System Voice` sentinel; never construct `SayVoice` from a name outside the list (RS-2).
- Linux/Windows: `FindPiperVoice(key)` is true (a pure `os.Stat`, `install.go:148-160`) — **find-only**; the
  alert path never installs synchronously (FR-9).

`Resolution{Role, Requested, Spoken, Link (role|group|root|default), Reason}` is what `[S]` prints (the on-air
chip was dropped, MVS-D-24). `Spoken` is the only name ever displayed or spoken (RS-18).

## 4. The fallback matrix (M5) — the table-driven test

Rows = every node incl. the root × states:

| State | Expected |
|---|---|
| assigned, installed/discovered | speaks the assigned voice; `Link = role` |
| assigned, **not installed** (Piper) | speaks the nearest resolvable ancestor; `[S]` "installing"; background install (FR-9) |
| assigned, **unknown** (macOS name not in the list) | ancestor; `[S]` "unknown on this host" |
| assigned in the **other OS's** table only | ancestor (this host's table is empty for the node) |
| list **not yet discovered** (first seconds) | the curated list answers: a curated name resolves, any other falls back to the ancestor with `[S]` "unknown on this host" until the list lands (D-R2-2 — no trust window) |
| the **sentinel** `System Voice` | always resolves; the identity lines say "your correspondent" when it has no name |
| inherited (empty) | the parent's resolution, `Link = group|root` |
| root unresolvable | platform default (System Voice / first installed / catalogue default); `[S]` states it |
| Linux, **nothing installed** | visual-only takeover with the *installing* state — the one legitimately silent row |

## 4b. The tone classes (FR-11; MVS-D-26/28)

`cast.Class` = **disaster** · **warning** · **watch** · **advisory** · **statement** · **storm** — the keys; which
products fall in each is `02-analysis/tones.md` §2's (the rule of record; not restated here). `ToneName(class)`: disaster and warning → `dual-tone`; watch → `1050`;
advisory → `classic`; statement → `soft-chime`; storm → `low-sweep`. `Muted(class, tones) bool` = `Mode == "mute"`
and the class is in `Muted`. A muted class sounds no tone; the words always read (MVS-D-26).

## 5. Segment and job tagging (FR-3)

`Segment{Key, Text, Pause, Role}` — `Role` is a synth-local enum (`weather` · `maritime` · `fire` · `seismic` ·
`station`), resolved at render by a `resolve func(Role) synth.Voice` injected into `Source`; the PCM cache key is
`voiceName + "\x00" + seg.Key`; `renderedSeg` carries the voice that rendered it. **The hand-over is not a
segment**: the Source decides it when the next segment's resolved voice differs from the voice that last spoke,
and speaks one injected `handoff(from, to)` line (script-tree backed) as a single `Say` (red-team B-3). A
**resolution generation** on the Source, bumped by any assignment change, lets `play`'s chunk loop re-resolve
only on a bump — the resolver is find-only and never runs under the install mutex (red-team B-7). Narration jobs
carry a role (`breaking` · `severe_read`) on `narrationJob`; the Station Director (the renamed arbiter) forwards
it to `tone`/`render` and never reads a voice itself.

## 6. Fixtures (NFR-5, FR-8)

`platform/config/testdata/`: `0.13.0-only.toml` (no tables) · `0.14.0-both-os.toml` · `0.14.0-macos-only.toml` ·
`0.14.0-piper-only.toml` · `unknown-key.toml` (`wether = "Amy"`) · `wrong-type.toml` (`alerts = 3`) ·
`bad-tone.toml` (`watch = "loud"`) · `hostile-name.toml` (an SGR escape and a newline in a name).
