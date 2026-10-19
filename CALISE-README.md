# General notes (organise later)
- modified run_local.sh to use core.exe for windows 

# Notes from each Task
### **Task 1**
Some basic assumptions made: default to visible / hidden = false, Not an optional field. Additionally assuming that this field should be added to SportEvent given the context of wanting it returned by "GetSportEvent", rather than adding at a higher level on the Event itself. Also learnt you need to refresh the .proto file in for the Postman Service Definition.

### **Task 2**
- Moved `Hidden` from `SportEvent` to the top level of `Event` (field 8). Racing events need it too, and Task 5 searches every event by display, so it belongs on the event rather than on each sport/racing substructure. This revises the Task 1 placement above.
- Added `RaceEvent` (`Event.RacingData`, field 9) with `RaceType` (Horse / Greyhound / Harness enum), `Venue`, `RaceNumber`, `DistanceMetres`, `TrackCondition`, `Weather`, `Country` and `RaceClass`. All use the `Optional*` wrappers so partial updates merge like the existing fields.
- Added merger functions `MergeRaceType`, `MergeOptionalRaceType` and `MergeRaceEvent`, and wired `Hidden` and `RacingData` into `MergeEvent`.
- Added the `GetRacingEvent` RPC returning a flattened `core.RacingEvent`. Shared fields (ID, Name, StartTime, BettingStatus, Markets, Hidden) are converted in one helper used by both `SportEvent` and `RacingEvent` to reduce duplicate code.
- Added `racingtransform`: sets `RaceType` from `EventTypeID` (`horse_racing`, `greyhound_racing`, `harness_racing`) and builds a name such as "Randwick R3" from venue and race number when no name was supplied. It only fills in missing values, so re-running it is a no-op.
- Example payloads: `exampledata/03_new_race.json` and `exampledata/04_race_track_update.json`.

### **Task 3**
- Added `ClosedAt` to `Market` as `OptionalInt64` (unix nanos), matching `Market.StartTime` and `Event.StartTime` in the model. Ints keep comparisons simple for date searches; formatting as a string belongs in the response layer.
- Merged in `MergeMarket` with `MergeOptionalInt64`, so a partial update that doesn't mention `ClosedAt` leaves it alone.
- Added `markettransform` (registered after the sport and racing transforms). For each market that is `BettingClosed` with no `ClosedAt`, it returns a delta containing only the market `ID` and `ClosedAt`. It is its own transform because markets exist on both sport and racing events. The clock is injectable so tests use a fixed time.
- `Update` strips any `ClosedAt` sent by a client before merging (`stripClientClosedAt`), so the transform is the only thing that sets it.
- Markets are returned as raw `model.Market` by `GetSportEvent` / `GetRacingEvent`, so `ClosedAt` comes back as `{"Value": <nanos>}`, the same as market `StartTime`. A `core.Market` view with string times would be a possible follow-up.
- Example payload: `exampledata/05_close_market.json`.

# Assumptions
- **Plain bool merge (pinned, not yet confirmed as intended):** `MergeEvent` always takes `Hidden` from the right operand, because a plain bool cannot tell "not sent" apart from `false`. So any update or transform delta that leaves `Hidden` out makes the event visible again. All transforms copy `Hidden` from the full model into their deltas to avoid this, and clients must resend `Hidden` on every update. `TestMergeEvent_HiddenTakenFromRight` pins the current behaviour; revisit if partial updates should leave `Hidden` untouched.
- Racing event types are identified by `EventTypeID` values `horse_racing`, `greyhound_racing` and `harness_racing`.
- **`ClosedAt` records the first close only.** If a market closes, reopens and closes again, `ClosedAt` keeps the first close time and is never overwritten.
- **Only a market's own `BettingStatus` sets `ClosedAt`.** Closing the event (`Event.BettingStatus = BettingClosed`) does not set `ClosedAt` on its markets if their own status is still open or suspended.
- **`ClosedAt` is not accepted from requests.** It is treated as an audit field derived from the betting status, so any value a client sends is discarded in `Update`.
- **`ClosedAt` uses server time** at the moment the transform sees the close, because updates don't carry a timestamp.

# Decisions
- `GetRacingEvent` returns an empty response (no error) both when the event does not exist and when it exists but is not a race (`RacingData` is nil). This matches how `GetSportEvent` handles a missing event.
- Runner level data (barrier, jockey, ect) was left out of scope for Task 2.

# AI Usage
### Claude chat:
Learning resource for understanding proto basics. Used for troubleshooting environment setup (Linux vs Windows).
### Claude Code:
Task 1: diagnosed why `Hidden` was not persisting (transform delta resetting it, stale Postman service definition). Task 2: implemented the changes above from an outline I reviewed and adjusted. Task 3: implemented from an outline, with the `ClosedAt` behaviour decisions made by me.

# TODO (time permitting)
- extend test cases + make test data more comprehensive. Maybe use provided exampledata ? 
- Export and attach postman example requests to make demoing/manual testing easier