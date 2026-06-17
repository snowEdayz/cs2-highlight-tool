# Demo Round Tick Validation

## Input

* Demo: `9210250352818846092.dem`
* Parser: `github.com/markus-wa/demoinfocs-golang/v5@v5.2.0`
* Probe: `research/demo_round_probe.go`
* Command:

```bash
env GOCACHE=/private/tmp/codex-go-cache /opt/homebrew/bin/go run .trellis/tasks/06-16-full-round-player-pov-recording/research/demo_round_probe.go 9210250352818846092.dem monesy
```

## Result

The library can extract the core data required for full-round first-person POV recording:

* `events.RoundStart` gives round start tick.
* `events.RoundFreezetimeEnd` gives the required recording start tick because full-round POV recording should exclude freeze time.
* `events.RoundEnd` gives the safe round cutoff tick.
* `events.RoundEndOfficial` is useful when present, but is not guaranteed for the last round.
* `events.Kill` identifies target-player death by `Victim.SteamID64` and gives the death tick.
* `GameState().Participants().Playing()` at round/freeze time can map target SteamID to the `spec_player` slot.

For target `monesy` (`steam=76561199605406701`), all 20 rounds produced usable recording ranges. The target slot was `12` in every round.

## Simulation Summary

```text
round=01 target_slot=12 record=1181->2132   end_by=target_death
round=02 target_slot=12 record=4248->4977   end_by=target_death
round=03 target_slot=12 record=6427->9903   end_by=target_death
round=04 target_slot=12 record=12140->17319 end_by=round_end
round=05 target_slot=12 record=18535->24951 end_by=target_death
round=06 target_slot=12 record=26291->31817 end_by=round_end
round=07 target_slot=12 record=33033->36676 end_by=target_death
round=08 target_slot=12 record=39129->45183 end_by=target_death
round=09 target_slot=12 record=47187->47968 end_by=target_death
round=10 target_slot=12 record=50554->54391 end_by=target_death
round=11 target_slot=12 record=56076->59761 end_by=target_death
round=12 target_slot=12 record=61615->64788 end_by=target_death
round=13 target_slot=12 record=66579->69282 end_by=target_death
round=14 target_slot=12 record=71179->74027 end_by=target_death
round=15 target_slot=12 record=75933->80885 end_by=round_end
round=16 target_slot=12 record=82101->87134 end_by=target_death
round=17 target_slot=12 record=90527->94363 end_by=target_death
round=18 target_slot=12 record=95579->98408 end_by=target_death
round=19 target_slot=12 record=99624->104570 end_by=target_death
round=20 target_slot=12 record=105786->107156 end_by=target_death
```

## Edge Cases Found

* Round 1 had `RoundStart` tick `0`; `RoundFreezetimeEnd` was valid (`1181`). This supports using freeze end as the recording start so freeze time is not included.
* Round 20 had no `RoundEndOfficial` tick, but had `RoundEnd=110099`. The implementation should treat `RoundEnd` as the primary safe cutoff and use `RoundEndOfficial` only as extra metadata or fallback.
* In a second run with the top-fragging player, a target death occurred after `RoundEnd`. The implementation must ignore target deaths outside `[record_start, round_end]` and end that round at `RoundEnd`.

## Implementation Implications

* Add a demo parser output for full-round POV candidates: per round, expose `round`, `start_tick`, `freeze_end_tick`, `end_tick`, optional `official_end_tick`, `target_slot`, and optional `death_tick`.
* Recording segments should use `record_start = freeze_end_tick`, fallback to `start_tick` only if freeze end is missing.
* Recording segments should use `record_end = death_tick` only when the death tick is within `[record_start, round_end]`; otherwise use `round_end`.
* The recording generator can produce one take per round, then append existing victim clip passes after all full-round POV passes.
