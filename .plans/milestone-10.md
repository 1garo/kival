# Implementation Plan for Milestone 10 - Compaction (Merge)

## Overview

**Objective**: Implement log compaction to reclaim disk space by removing dead records (old versions and tombstones)
**Package**: `github.com/1garo/kival/kv`
**Related Package**: `github.com/1garo/kival/log`
**Goal**: Consolidate multiple log files into a single compacted file containing only live key-value pairs

## Background

### The Problem
- **Dead Bytes**: Old versions of keys remain in older log files after updates
- **Tombstones**: Deleted keys stay in files forever
- **Unbounded Growth**: Disk usage increases while live data stays constant
- **Read Amplification**: Multiple files need to be checked during startup index rebuild

### Bitcask Compaction Strategy
Bitcask uses a **merge** process that:
1. Scans all non-active (read-only) log files
2. Builds a temporary index of only the latest version of each key
3. Writes live records to a new compacted file
4. Atomically replaces old files with the compacted one
5. Updates the in-memory index with new positions

## Current State Analysis

### Existing Infrastructure
- **KV Store**: `kv/kv.go` - Manages keyDir (in-memory index) and log rotation
- **Log Files**: `log/log.go` - Manages individual log segments
- **Index**: `map[string]log.LogPosition` - Maps keys to file locations
- **Log Rotation**: Already implemented (Milestone 8)
- **Tombstones**: Already implemented (Milestone 7)

### Current File Management
```
Data Directory Structure:
  1.data (read-only)
  2.data (read-only)
  3.data (active)
```

**Key Components**:
- `activeLog`: Currently writable log file
- `logs`: Map of `map[uint32]log.Log` (read-only segments)
- `keyDir`: In-memory index of live keys

### Data Flow
```
Put(key, value)
  -> Append to activeLog
  -> Update keyDir[key] = newPosition

Get(key)
  -> Lookup keyDir[key]
  -> Read from appropriate log file

Delete(key)
  -> Append tombstone record
  -> Remove from keyDir
```

## Implementation Plan

### Phase 1: Compaction Triggering Strategy (Priority: HIGH)

#### 1.1 Define Trigger Conditions
```go
const (
    // Trigger compaction when dead bytes exceed 50% of total
    DefaultCompactionThreshold = 0.5
    
    // Or trigger after N writes
    DefaultCompactionInterval = 1000
)
```

**Trigger Options**:
1. **Threshold-based**: When `deadBytes / totalBytes > threshold`
2. **Interval-based**: Every N write operations
3. **Manual**: Explicit `Compact()` API call
4. **Startup**: Optional compaction on database open

#### 1.2 Add Metrics Tracking
```go
type kv struct {
    // ... existing fields ...
    
    // Compaction metrics
    totalBytes      int64
    liveBytes       int64
    writeCount      uint64
    lastCompaction  time.Time
}
```

**Metrics to Track**:
- Total bytes in all log files
- Live bytes (sum of latest record sizes)
- Dead bytes ratio: `(total - live) / total`
- Write operation count since last compaction

#### 1.3 Implement Threshold Check
```go
func (k *kv) shouldCompact() bool
func (k *kv) updateMetrics(pos log.LogPosition, isUpdate bool)
```

### Phase 2: Compaction Core Algorithm (Priority: CRITICAL)

#### 2.1 Compaction Process Overview
```go
func (k *kv) Compact() error
```

**Steps**:
1. **Lock**: Acquire exclusive write lock (no concurrent writes during compaction)
2. **Collect**: Gather all read-only log files eligible for compaction
3. **Iterate**: Scan keyDir to get all live key-value pairs
4. **Write**: Create new compacted file, write live records sequentially
5. **Sync**: Ensure new file is fully synced to disk
6. **Swap**: Atomically replace old files with compacted file
7. **Update**: Rebuild keyDir with new positions
8. **Cleanup**: Delete old log files
9. **Unlock**: Release write lock

#### 2.2 File Selection Strategy
```go
func (k *kv) selectFilesForCompaction() []uint32
```

**Selection Criteria**:
- All read-only files (not the active log)
- Or files with high dead bytes ratio
- Or files older than a certain age
- Or all files if doing full compaction

**Decision**: Start with **all read-only files** for simplicity

#### 2.3 Live Record Collection
```go
func (k *kv) collectLiveRecords() ([]compactionEntry, error)

type compactionEntry struct {
    key   []byte
    value []byte
    // Keep original timestamp for ordering?
}
```

**Why Scan keyDir?**
- keyDir already contains only live keys (tombstones removed them)
- O(N) where N = number of live keys
- No need to scan entire log files
- Guaranteed to have latest version of each key

#### 2.4 Writing Compacted File
```go
func (k *kv) writeCompactedFile(entries []compactionEntry) (string, error)
```

**Process**:
1. Create temporary file: `compact.tmp`
2. For each entry:
   - Read value from current location (via keyDir)
   - Append to compact.tmp
   - Record new position
3. Sync compact.tmp to disk
4. Return new positions map

**Naming Convention**:
- Temporary: `compact.tmp`
- Final: Will be renamed to `1.data` (resetting file IDs)

### Phase 3: Atomic File Swap (Priority: CRITICAL)

#### 3.1 Crash-Safe Swap Strategy
```go
func (k *kv) performAtomicSwap(
    compactedFile string,
    oldFiles []uint32,
    newPositions map[string]log.LogPosition,
) error
```

**Atomic Swap Steps**:
1. Write **compaction marker** file: `COMPACTING`
   ```json
   {
     "old_files": [1, 2],
     "compacted_file": "compact.tmp",
     "timestamp": 1234567890
   }
   ```
2. Close all old log files
3. Rename `compact.tmp` -> `1.data`
4. Update in-memory structures:
   - Clear `logs` map
   - Set new active log
   - Update `keyDir` with new positions
5. Delete old files: `2.data`, `3.data`, etc.
6. Delete `COMPACTING` marker

#### 3.2 Compaction Marker for Crash Recovery
```go
const compactionMarkerFile = "COMPACTING"

type compactionMarker struct {
    OldFiles      []uint32  `json:"old_files"`
    CompactedFile string    `json:"compacted_file"`
    Timestamp     int64     `json:"timestamp"`
}
```

**Purpose**: If crash occurs during compaction:
- On restart, check for `COMPACTING` marker
- If present, complete or rollback the compaction
- Ensures no data loss or corruption

#### 3.3 File ID Reset Strategy
**Decision**: Reset file IDs to start from 1

**Why?**
- Simplifies file management
- Avoids ever-increasing file IDs
- Compacted file becomes `1.data`
- Next rotation creates `2.data`, etc.

**Alternative**: Maintain sequential IDs (more complex, not needed)

### Phase 4: Crash Recovery Integration (Priority: HIGH)

#### 4.1 Recovery on Startup
Modify `kv.New()` to:
```go
func New(path string) (*kv, error) {
    // Check for incomplete compaction
    if marker, err := readCompactionMarker(path); err == nil {
        // Complete or rollback compaction
        if err := recoverCompaction(path, marker); err != nil {
            return nil, fmt.Errorf("compaction recovery failed: %w", err)
        }
    }
    
    // Continue with normal initialization...
}
```

#### 4.2 Recovery Scenarios

**Scenario 1: Compaction marker exists, compacted file exists**
- Compaction completed writing but marker not cleaned up
- Delete old files (if any remain)
- Delete marker
- Continue

**Scenario 2: Compaction marker exists, compacted file missing**
- Compaction failed mid-write
- Delete any partial compacted file
- Delete marker
- Old files are still valid, continue normally

**Scenario 3: Compaction marker missing, orphaned temp file exists**
- Crash after write but before marker creation
- Delete `compact.tmp` (it's incomplete)
- Continue normally

### Phase 5: Index Rebuilding (Priority: HIGH)

#### 5.1 Position Mapping
After compaction, all positions change:
- Old: `LogPosition{FileID: 2, ValuePos: 150, ValueSize: 10}`
- New: `LogPosition{FileID: 1, ValuePos: 75, ValueSize: 10}`

**Implementation**:
```go
// During writeCompactedFile, track new positions
newPositions := make(map[string]log.LogPosition)
offset := int64(0)

for _, entry := range entries {
    record := record.Encode(entry.key, entry.value)
    // Write at offset...
    
    newPositions[string(entry.key)] = log.LogPosition{
        FileID:    1, // New compacted file ID
        ValuePos:  offset,
        ValueSize: uint32(len(entry.value)),
    }
    offset += int64(len(record))
}
```

#### 5.2 Index Update Strategy
```go
// Replace entire keyDir atomically
k.keyDir = newPositions

// Or update each entry
for key, pos := range newPositions {
    k.keyDir[key] = pos
}
```

### Phase 6: API and Configuration (Priority: MEDIUM)

#### 6.1 Public API
```go
// KV interface extension
type KV interface {
    Put(key []byte, data []byte) error
    Get(key []byte) ([]byte, error)
    Del(key []byte) error
    
    // NEW: Manual compaction trigger
    Compact() error
    
    // NEW: Get compaction statistics
    Stats() CompactionStats
}

type CompactionStats struct {
    TotalBytes     int64
    LiveBytes      int64
    DeadBytes      int64
    DeadRatio      float64
    LastCompaction time.Time
    TotalCompactions uint64
}
```

#### 6.2 Configuration Options
```go
type Config struct {
    // Existing options...
    
    // Compaction settings
    EnableAutoCompaction bool
    CompactionThreshold  float64  // 0.0 - 1.0
    CompactionInterval   int      // writes between compactions
}

func NewWithConfig(path string, config Config) (*kv, error)
```

#### 6.3 Background Compaction (Optional Future Enhancement)
```go
// For later: Run compaction in background goroutine
func (k *kv) StartBackgroundCompaction()
func (k *kv) StopBackgroundCompaction()
```

### Phase 7: Testing Strategy (Priority: CRITICAL)

#### 7.1 Unit Tests
```go
// Basic compaction
func TestCompact_RemovesDeadRecords(t *testing.T)
func TestCompact_PreservesLiveRecords(t *testing.T)
func TestCompact_RemovesTombstones(t *testing.T)

// File management
func TestCompact_CreatesSingleCompactedFile(t *testing.T)
func TestCompact_DeletesOldFiles(t *testing.T)
func TestCompact_ResetsFileIDs(t *testing.T)

// Index accuracy
func TestCompact_UpdatesKeyDirPositions(t *testing.T)
func TestCompact_MaintainsReadAccuracy(t *testing.T)
func TestCompact_AllOperationsAfterCompaction(t *testing.T)
```

#### 7.2 Edge Cases
```go
// Empty database
func TestCompact_EmptyDatabase(t *testing.T)

// Single record
func TestCompact_SingleRecord(t *testing.T)

// All records deleted
func TestCompact_AllRecordsDeleted(t *testing.T)

// Large values
func TestCompact_LargeValues(t *testing.T)

// Many small records
func TestCompact_ManySmallRecords(t *testing.T)
```

#### 7.3 Crash Recovery Tests
```go
// Crash during write
func TestCompactRecovery_CrashDuringWrite(t *testing.T)

// Crash after write before swap
func TestCompactRecovery_CrashBeforeSwap(t *testing.T)

// Crash after swap before cleanup
func TestCompactRecovery_CrashBeforeCleanup(t *testing.T)

// Power loss at various stages
func TestCompactRecovery_PowerLossScenarios(t *testing.T)
```

#### 7.4 Integration Tests
```go
// Full workflow
func TestCompact_FullWorkflow(t *testing.T)

// Concurrent operations (if supported)
func TestCompact_ConcurrentOperations(t *testing.T)

// Multiple compactions
func TestCompact_MultipleCompactions(t *testing.T)

// Metrics accuracy
func TestCompact_MetricsAccuracy(t *testing.T)
```

## Implementation Phases Timeline

### Phase 1: Foundation (Days 1-2)
- [ ] Add metrics tracking to kv struct
- [ ] Implement `shouldCompact()` logic
- [ ] Add configuration options
- [ ] Write initial unit tests for metrics

### Phase 2: Core Compaction (Days 3-4)
- [ ] Implement `Compact()` method
- [ ] Implement file selection
- [ ] Implement live record collection via keyDir
- [ ] Implement compacted file writing
- [ ] Write tests for core compaction logic

### Phase 3: Atomic Swap (Days 5-6)
- [ ] Implement compaction marker
- [ ] Implement atomic file swap
- [ ] Implement file ID reset
- [ ] Write tests for atomic operations

### Phase 4: Crash Recovery (Days 7-8)
- [ ] Modify `kv.New()` to check for markers
- [ ] Implement recovery logic
- [ ] Test all crash scenarios
- [ ] Verify data integrity after recovery

### Phase 5: Integration (Days 9-10)
- [ ] Integrate with existing KV operations
- [ ] Add auto-compaction triggers (optional)
- [ ] Add public API methods
- [ ] Write comprehensive integration tests

### Phase 6: Polish (Days 11-12)
- [ ] Add detailed logging
- [ ] Optimize performance if needed
- [ ] Add benchmarks
- [ ] Final review and cleanup

## Key Design Decisions

### 1. **Scan keyDir, Not Files**
- **Decision**: Use in-memory index to find live records
- **Rationale**: O(N) live keys vs O(M) total records, no duplicates to handle
- **Trade-off**: Requires all keys to fit in memory (already a Bitcask constraint)

### 2. **Reset File IDs**
- **Decision**: Start from 1 after compaction
- **Rationale**: Simplifies file management, prevents ID overflow
- **Trade-off**: Breaks continuity, but IDs are internal

### 3. **Compact All Read-Only Files**
- **Decision**: Don't be selective initially
- **Rationale**: Simpler implementation, guaranteed to remove all dead bytes
- **Trade-off**: More I/O than necessary, can optimize later

### 4. **Synchronous Compaction**
- **Decision**: Block writes during compaction (initially)
- **Rationale**: Simpler, ensures consistency
- **Trade-off**: Availability impact, can add background later

### 5. **Compaction Marker File**
- **Decision**: Use JSON file for crash detection
- **Rationale**: Human-readable, easy to debug
- **Trade-off**: Slightly larger than binary, negligible

## Error Handling Strategy

### Compaction Failures
```go
var (
    ErrCompactionFailed = errors.New("compaction failed")
    ErrCompactionInProgress = errors.New("compaction already in progress")
    ErrNothingToCompact = errors.New("no files to compact")
)
```

### Recovery
- Always leave database in valid state
- Incomplete compaction can be resumed or rolled back
- Never lose live data
- Marker file ensures idempotency

## Performance Considerations

### I/O Optimization
- Read live records in batch if possible
- Use buffered I/O for writing
- Sync only at end (not per record)

### Memory Usage
- Stream records instead of loading all into memory
- Process in chunks if key count is very large

### Locking Strategy
- Exclusive lock during entire compaction
- Short lock for auto-compaction check
- Consider read-write lock for future background compaction

## Files to Modify

### Primary Changes
1. **kv/kv.go**
   - Add metrics fields to kv struct
   - Add `Compact()` method
   - Add `Stats()` method
   - Modify `New()` for recovery
   - Add helper methods

2. **kv/compaction.go** (new file)
   - Compaction logic
   - File selection
   - Atomic swap
   - Recovery functions

3. **kv/config.go** (new file, optional)
   - Configuration struct
   - Default values

### Supporting Changes
4. **log/log.go** (minor)
   - May need `Delete()` method for file cleanup
   - May need `GetPath()` for file operations

5. **kv/kv_test.go**
   - Add comprehensive compaction tests
   - Add crash recovery tests

## Success Criteria

### Functional
- [ ] Compaction removes all dead records
- [ ] Compaction preserves all live records
- [ ] Index remains accurate after compaction
- [ ] Reads work correctly after compaction
- [ ] Writes work correctly after compaction
- [ ] Crash recovery handles all scenarios

### Performance
- [ ] Compaction completes in reasonable time
- [ ] Database remains usable during/after compaction
- [ ] Disk space is reclaimed
- [ ] No memory leaks

### Quality
- [ ] ≥ 85% test coverage for compaction code
- [ ] All edge cases tested
- [ ] Crash scenarios tested
- [ ] Clear error messages
- [ ] Documentation complete

## References

### Bitcask Paper Concepts
- Merge process described in Section 4.2
- Crash recovery strategy
- File management

### Related Code
- `kv/kv.go` - Current KV implementation
- `log/log.go` - Log file management
- `record/record.go` - Record encoding/decoding

### Testing Patterns
- See `log/log_integration_test.go` for table-driven tests
- Use `t.TempDir()` for isolation
- Use `t.Cleanup()` for resource cleanup

## Open Questions

1. **Should compaction be automatic or manual only?**
   - Recommendation: Start with manual, add auto later

2. **Should we support partial compaction (selective files)?**
   - Recommendation: Start with full compaction, optimize later

3. **Should compaction run in background?**
   - Recommendation: Start synchronous, add background later

4. **What compaction threshold should be default?**
   - Recommendation: 50% dead bytes

## Next Steps

1. **Review this plan** - Ensure all scenarios covered
2. **Decide on open questions** - Automatic vs manual, etc.
3. **Start Phase 1** - Add metrics and configuration
4. **Iterative development** - One phase at a time with tests

---

*This plan provides a comprehensive roadmap for implementing Bitcask-style compaction while maintaining data integrity and crash safety.*
