# Plan: Full Compaction
Compaction reads all existing log files, writes only the latest live version of each key to a new compacted log, then replaces the old files.

## Steps

### Step 1: Add Merge() method to KV interface
- New method: Merge() error
- Add to kv/kv.go

### Step 2: Implement merge logic in kv/kv.go
1. Create new log file with next ID (activeLog.ID() + 1)
2. Iterate through keyDir (in-memory index)
3. For each key:
   a. Read value from current position (using logs map or activeLog)
   b. Append to new compacted log
   c. Update keyDir with new positions
4. Mark all old logs as read-only
5. Set new compacted log as active
6. Close and delete old log files

### Step 3: Handle file deletion
- Close old log files
- Use os.Remove() to delete old .data files
- Clean up logs map

## Key considerations
- Tombstones are NOT written to compacted log (they're already removed from keyDir)
- Keep timestamps to determine "latest" version (already handled by keyDir)
- Need method to get key+value from LogPosition (you have ReadAt already)

Suggested test cases
1. Compact with all live keys
2. Compact with some deleted keys (tombstones)
3. Compact with multiple log files
4. Verify old files are deleted
