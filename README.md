# Kival - A Bitcask-Inspired Key-Value Store (Go)

A learning-focused, Bitcask-inspired key-value store implemented in Go.

This project explores how log-structured storage engines work under the hood: append-only logs, in-memory indexes, crash recovery, and data integrity.

The goal is **correctness first**, then **performance**, while keeping the implementation small, inspectable, and educational.

---

## Project Goals

- Understand append-only storage engines
- Learn how Bitcask-style indexing works
- Practice crash recovery and corruption handling
- Build a minimal but correct key-value store in Go

### Non-goals (for now)

- Networking
- Distribution
- Transactions
- Encryption or compression

---

## Record Format

Each record is stored sequentially in a `.data` file using the following layout:

| CRC (4 bytes) | KeySize (4 bytes) | ValueSize (4 bytes) | Key | Value |

Details:
- CRC32 is used to protect against corruption
- All integer fields use little-endian encoding
- Keys and values are stored as raw byte slices

---

## Completed Milestones

### Milestone 1 — Append-Only Log File

- Single active `.data` file
- Records appended sequentially using `WriteAt`
- Explicit `writePos` tracking
- No in-place updates

Outcome: Durable, sequential writes.

---

### Milestone 2 — LogPosition Abstraction

LogPosition structure:
- FileID
- Offset

Purpose:
- Stored as the value in the in-memory index
- Enables constant-time reads

---

### Milestone 3 — In-Memory Index

- Implemented as `map[string]LogPosition`
- Updated on every `Set`
- Always points to the latest version of a key

Outcome: Fast reads without scanning the log.

---

### Milestone 4 — Record Encode / Decode

- Binary encoding using little-endian format
- CRC32 checksum generation and validation
- Decode logic validates:
  - File boundaries
  - Partial writes
  - CRC mismatches

Outcome: Safe, verifiable record reads.

---

### Milestone 5 — Crash Recovery via Index Rebuild

- On startup:
  - Sequentially scan the log file
  - Decode records
  - Rebuild the in-memory index
- Stop scanning on partial or corrupt records

Outcome: Crash-safe recovery without WAL replay.

---

### Milestone 6 — Get / Set API

Public API:
- `Set(key, value)`
- `Get(key)`

Behavior:
- Set appends an encoded record
- Get uses the in-memory index plus `ReadAt`
- No log scanning during reads

---

## Upcoming Milestones (1–2 Days Each)

Each milestone is intentionally small to allow fast feedback loops.

---

### Milestone 7 — Delete Support (Tombstones)

Goal: Implement `Delete(key)`

Details:
- Store a tombstone record
- Same record format
- ValueSize set to 0
- During recovery:
  - Remove key from index when tombstone is found
- Get returns `ErrNotFound`

- [x] Done

Why: Required for correctness before compaction.

---

### Milestone 8 — Multiple Data Files (Log Rotation)

Goal: Support more than one `.data` file

Details:
- Introduce a max file size
- When the active log exceeds the limit:
  - Close it
  - Create a new active log
- Index entries include FileID

- [x] Done
  - [x] Test if ensure capacity will create new file correctly.

Why: Enables compaction and bounds file sizes.

---

### Milestone 8.1 — Added tests
Goal: Add tests

Packages:
  - [ ] kv
  - [ ] log
  - [ ] record


- [ ] Done


---

### Milestone 9 — Reads from Older Log Files

Goal: Read values from non-active log files

Details:
- Maintain a map of FileID to logFile
- Get reads from the correct log using FileID

- [x] Done

Why: Required once log rotation exists.

---

### Milestone 10 — Compaction (Merge)

Goal: Reclaim disk space

Details:
- Create a new compacted log
- Write only the latest live version of each key
- Atomically swap logs
- Rebuild the index

Why: Core Bitcask feature.

---

### Milestone 11 — File Sync & Durability Controls

Goal: Improve durability semantics

Details:
- Configurable fsync strategy:
  - Always
  - Every N writes
  - Never (benchmark mode)

Why: Teaches durability versus performance trade-offs.

---

### Milestone 12 — Basic Metrics & Introspection

Goal: Improve observability

Track:
- Log size
- Number of keys
- Dead bytes ratio
- Last compaction time

Why: Makes system behavior visible and debuggable.

---

## Testing Strategy

- Unit tests for:
  - Encode / Decode logic
  - CRC corruption detection
  - Partial write handling
- Crash simulation:
  - Kill the process mid-write
  - Restart and rebuild the index

---

## What This Project Teaches

- Why append-only logs scale
- Why indexes must be rebuilt on startup
- How real databases detect and handle corruption
- Why deletes are harder than writes
- How compaction works in practice

---

## Final Note

This is **not a toy project**.

The design is inspired by:
- Bitcask
- LevelDB log layers
- Kafka log segments
- Write-Ahead Logs (WAL)

Completing all milestones results in a real storage engine core.

---

Possible next steps:
- Turn this into a blog or learning series
- Add benchmarks
- Compare with LevelDB or Pebble
- Evolve it into a production-ready embedded KV store
