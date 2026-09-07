# Agent UAT — P1 (the foundation), multi-voice-support 0.14.0

**This is an AGENT/developer check, not end-user UAT** (HUM LEAD, 2026-08-30). It exercises the config file
directly, which is not how a listener sets a cast — they will use Setup, and Setup lands at **P4**. Nothing in
here should be read as "the user journey"; it is the internal guard rail that stops an engineer, an agent or a
sync conflict putting a weird, non-standard or half-written value into the file during development.

**Where the real end-user UAT starts:** P4, `07-readiness/uat-p4.md` — open Setup, pick a voice for the alerts
and a voice for the reports, preview them, save, and listen. Until then there is no listener-facing surface for
this feature at all, because nothing below the root voice is read before P2 and nothing draws it before P4.

**Why the file-level guards still earn their place** (and are not merely developer hygiene):

- `~/.config/watchpost/config.toml` is a plain file the README documents by path, with a whole *"For tinkerers"*
  section that hands the reader TOML to paste (`[fire]` thresholds, `[keys]`, `[providers.<name>]`). Hand-editing
  is a supported path here, not an unsupported one.
- The NFR-5 preservation is not about a listener typing something wrong. It is about **one config, two binaries**:
  a dotfile-synced or shared home directory where one machine is on 0.14.0 and the other is still on 0.13.0, or a
  downgrade after a bad release. The older binary does not need to *offer* the cast to destroy it — it only needs
  to **save once**, for any reason (a theme change, a Setup save), and every key it does not know is gone.
  Cross-machine sync is already an assumption of this design: it is why a role's pair carries both the macOS and
  the Piper half, and why `0.14.0-both-os.toml` is a fixture.

So: run this as a regression guard on the file contract. Judge the feature at P4.

---

**Roughly ten minutes.** Nothing here touches your real config: every step runs against a throwaway
`XDG_CONFIG_HOME`, and step 0 is the command that makes it.

**What P1 can and cannot show.** P1 is the foundation, so most of it is deliberately invisible: the cast is
*parsed, validated and preserved*, but **nothing speaks it until P2**, and the `[S]` diagnostics table and the
Setup pickers are **P4**. What is being checked here is:

1. nothing about today's radio changed (FR-2 — the anchor every later batch is measured against);
2. a hand-written cast is accepted rather than rejected or quietly deleted;
3. a bad value is named in words a person can act on;
4. **a file this build does not fully understand survives a save** — the one-config-two-binaries guarantee.

---

## 0. Set up a sandbox (does not touch `~/.config/watchpost`)

```sh
cd ~/Desktop/PERSONAL_PROJECTS/watchpost
make build

export UAT=$(mktemp -d)/cfg && mkdir -p "$UAT/watchpost"
cat > "$UAT/watchpost/config.toml" <<'EOF'
# A 0.13.0-shaped file, hand-edited with a 0.14.0 cast, plus two keys from an
# imaginary future build that this binary knows nothing about.
theme = "quattro-dark"
voice = "Samantha"
ticker_muted = true
future_top_level = "keep me"

[[locations]]
label = "Oceanside, CA"
lat = 33.1959
lon = -117.3795
tag = "OCNSD"

[radio]
mode = "synth"
cast = "cast"
future_radio_key = "keep me too"

[radio.voices.alerts]
macos = "Rishi"
piper = "en_US-ryan-medium"

[radio.voices.fire]
macos = "Nobody At All"

[radio.tones]
mode = "mute"
muted = ["watch", "advisory", "not-a-class"]
EOF
cp "$UAT/watchpost/config.toml" "$UAT/before.toml"
```

Every command below is prefixed `XDG_CONFIG_HOME="$UAT"`. Drop that prefix and you are editing your real config.

---

## 1. The cast is accepted, not rejected

```sh
XDG_CONFIG_HOME="$UAT" ./dist/watchpost report --report-only "Oceanside, CA"
```

**Pass:** a normal report. No error, no complaint about `[radio.voices.*]`, `cast = "cast"`, the unknown
`future_*` keys, or the bogus `"not-a-class"` tone entry.

**Why it matters:** a hand-edited cast — or a file written by a *newer* build — must not stop the app.

---

## 2. Today's radio is unchanged (FR-2)

```sh
XDG_CONFIG_HOME="$UAT" ./dist/watchpost
```

Tune the radio (`space`) and listen to a full cycle.

**Pass, by ear and by eye:**

- One voice reads everything — **Samantha**, the root — even though the file assigns Rishi to *alerts* and a
  nonsense name to *fire*. Nothing below the root is read yet; that is P2.
- The segment order, the wording and the pauses are exactly 0.13.0's.
- The sign-off still says *"This is Samantha for Watchpost Weather Radio…"*.
- The visualizer, `[m]`, the volume keys and the ticker all behave as they did.

**Fail:** any change in what is said, the order it is said in, or how long the gaps are.

---

## 3. A bad value is named in words you can act on

```sh
XDG_CONFIG_HOME="$UAT" sed -i '' 's/cast = "cast"/cast = "casr"/' "$UAT/watchpost/config.toml"
XDG_CONFIG_HOME="$UAT" ./dist/watchpost report --report-only "Oceanside, CA"
```

**Pass:** it refuses to start and says

> `[radio] cast must be "" (single voice) or "cast" (got "casr") — fix it or move it aside and rerun 'watchpost setup'`

Try the same with `mode = "muted"` under `[radio.tones]`; you should get the equivalent line naming
`[radio.tones] mode`. Then put `cast = "cast"` back.

**Judge it as a reader, not as a tester:** does the message tell you *what is wrong*, *where*, and *what to do*?
That is the standard, and it is yours to reject.

---

## 4. The important one — a save does not eat what it does not understand

Open the app and change the colour theme, which is the cheapest thing that writes the file:

```sh
XDG_CONFIG_HOME="$UAT" ./dist/watchpost
```

Press `t`, pick any theme, `enter`, then `q`. Now check that nothing was eaten:

```sh
for k in future_top_level future_radio_key 'Rishi' 'en_US-ryan-medium' \
         'Nobody At All' ticker_muted 'not-a-class'; do
  printf '%-24s ' "$k"
  grep -qF "$k" "$UAT/watchpost/config.toml" && echo PRESENT || echo '*** LOST ***'
done
```

**Pass:** all seven say `PRESENT`, and the theme line changed.

**Do not use a plain `diff` here** — or if you do, expect noise that is not a defect. Saving re-marshals the
whole document, so quoting flips from `"` to `'`, keys are reordered alphabetically, empty fields like `tz = ''`
appear, and **comments are lost**. None of that is new: `toml.Marshal` has always done it, in 0.13.0 too. The
grep above is what separates the signal from that.

A second save of an unchanged config is byte-for-byte identical to the first, so the churn happens once and then
settles. You can confirm it: run the app again, press `t`, pick the *same* theme, quit, and the file will not
have moved.

**Why this is the one that matters.** Before this batch, a save went through the typed struct and silently
dropped every key the struct had no field for. That means a 0.13.0 binary opening your 0.14.0 file would have
**deleted your entire cast** on its next write — you would lose every assignment by doing nothing more than
running an older build once. If anything in that diff vanished, P1 has not delivered its main promise.

**Known and deliberate:** a key inside `[[locations]]` or `[[recent]]` that this build does not know *is*
dropped. Array indices are not stable across a save that reorders them, and copying by index could attach a key
to the wrong location — a recorded limitation in NFR-5, not an oversight.

---

## 5. Your 0.13.0 mute carried over — with one accepted change

The sandbox file has `ticker_muted = true`, as a 0.13.0 file would.

```sh
grep -A3 '\[radio.tones\]' "$UAT/watchpost/config.toml"
grep ticker_muted "$UAT/watchpost/config.toml"
```

**Pass:** the file still carries `ticker_muted = true` after the save in step 4, so a 0.13.0 binary reading it
would still mute its ticker.

**The accepted change (MVS-D-34, E-6), which is yours to re-open if you dislike it in practice:** in 0.13.0 `[M]`
muted the tone *and* the words. From 0.14.0 it mutes **tones only** — the words always read. So on upgrade you
will hear breaking-news narration you used to have silenced. It is in the CHANGELOG, there is no legacy mapping,
and this is the moment to tell me if hearing it changes your mind.

---

## 6. Clean up

```sh
rm -rf "$(dirname "$UAT")" && unset UAT
```

---

## Recording the result

Pass or fail per step. A fail in **2** or **4** blocks P2. A complaint about the wording in **3** is a ruling on
error copy, not a bug.

**Not testable at P1, by design:** hearing a second correspondent, the hand-over lines, the per-class tones, the
`[S]` cast table, and the Setup pickers. Those are P2 through P4, and the listener-facing judgement belongs
there — see the note at the top.
