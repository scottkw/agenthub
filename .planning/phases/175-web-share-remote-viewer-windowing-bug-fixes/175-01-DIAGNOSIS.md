# 175-01 DIAGNOSIS — BUG-03 (#126) live timed reproduction

**Date:** 2026-07-08
**Operator:** Ken Scott (human-run live diagnostic; exit-detection poller runs in the Wails Go process, not the test surface)
**Gates:** 175-05 fix branch selection

## Procedure

Ran the mandatory timed live reproduction from 175-01-PLAN.md against a live app + real daemon,
recording tab-close outcomes for the exit-detection chain (`pollSessionStatus` → `session:exit` →
`App.tsx` handler → `handleCloseTab`).

## Observed runs

| # | Case | Elapsed (wall-clock) | Exited from inside session? | Tab auto-closed? | Result |
|---|------|----------------------|-----------------------------|------------------|--------|
| 1 | Baseline — unshared, immediate exit | immediate (~1–2s to close) | yes | **YES** | pass (expected) |
| 2 | Share-workflow — web-share enabled, session lived **> 5 minutes**, then exited | > 5 min | yes | **YES (auto-closed)** | pass — bug did **not** reproduce |
| 3 | Tie-breaker — shared, exited **< 5 minutes** | < 5 min | yes | **YES** | pass |

_Elapsed times are operator-reported; exact per-second capture was not recorded, but the deciding
variable — whether the tab auto-closed after crossing the 5-minute mark in the shared case — was
observed directly (case 2 closed normally)._

## Interpretation

The core prediction of the deadline hypothesis was: a shared session that lives **past the fixed
300 s exit-poll window** (`app.go:386` `deadline := time.Now().Add(300 * time.Second)`, non-renewing,
measured from poll start) would silently expire, emit no `session:exit`, and leave its tab open.

**That did not happen.** In case 2 the shared session lived well past 5 minutes and its tab still
auto-closed on exit. The baseline (case 1) and the <5-min shared tie-breaker (case 3) also closed
normally.

## VERDICT: **DISPROVED**

`app.go:386`'s fixed 300 s exit-poll deadline is **NOT** BUG-03's root cause — a shared session that
outlived the window still auto-closed its tab. BUG-03 did **not** reproduce in this session across
baseline, >5-min shared, or <5-min shared cases.

## Selected fix branch for 175-05

Take the **DISPROVED branch**: do **not** remove/re-arm the deadline as a blind behavioral fix.
Instead, add a **diagnostic-logging pass** around the exit-detection / tab-close path
(`handleCloseTab` / `ToggleWebServing` / `KillSession`, and the `session:exit` emit site) so that if
BUG-03 recurs in the field it is directly diagnosable. RESEARCH flags the un-timeout'd `http.Client`
at `internal/daemon/client.go:37` as a candidate hang site worth instrumenting.

**Note for verification (M-3x live UAT / 175-07):** BUG-03 was non-reproducible in this timed run.
The live-UAT gate for #126 should record this non-reproduction and treat the shipped instrumentation
(not a behavior change) as the deliverable, unless a reliable repro is later found.
