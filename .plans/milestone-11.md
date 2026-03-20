# Milestone 11 — File Sync & Durability Controls

## Goal
Add configurable fsync strategy to the log package with three modes: always, every N writes, and never.

## Files to Modify

1. `log/log.go` — Add sync strategy configuration to `logFile` and `New()`
2. `log/log_test.go` — Add tests for new sync behavior

## Steps

### Step 1: Define SyncStrategy type

In `log/log.go`, add a new type and constants:
- `type SyncStrategy int`
- `const Always SyncStrategy = iota` (default)
- `const EveryN SyncStrategy`
- `const Never SyncStrategy`

### Step 2: Add fields to logFile struct

Add to `logFile`:
- `syncStrategy SyncStrategy`
- `writeCount uint32` (for EveryN strategy)
- `syncEveryN uint32` (threshold for EveryN)

### Step 3: Modify `New()` function signature

Update `New()` to accept optional sync strategy. Suggested: add `SyncStrategy` and `SyncEveryN` parameters (simpler than options pattern).

### Step 4: Implement conditional sync in `Append()`

Replace unconditional `Sync()` call with logic:
- **Always**: Call `Sync()` every time
- **EveryN**: Call `Sync()` when `writeCount % syncEveryN == 0`
- **Never**: Never call `Sync()`

Increment `writeCount` after each append regardless of strategy.

### Step 5: Add tests

Add tests for each strategy:
- `TestAppend_AlwaysSync_SyncsEveryWrite`
- `TestAppend_EveryNSync_SyncsAtThreshold`
- `TestAppend_NeverSync_NeverCallsSync`
- Verify writeCount increments correctly

## Definition of Done

- [ ] `SyncStrategy` type and constants defined
- [ ] `New()` accepts sync strategy configuration
- [ ] `Append()` respects chosen strategy
- [ ] Tests pass for all three strategies
- [ ] Existing tests still pass (backward compatible)
- [ ] `go vet` and `go fmt` pass

## Notes

- Default behavior (Always) matches current implementation — no breaking changes
- "EveryN" is useful for benchmarks: e.g., sync every 1000 writes
- "Never" is for extreme performance scenarios where durability is handled externally
