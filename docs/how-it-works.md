# Kival Storage Model

This page explains how Kival opens a database, writes data, rotates log files, and compacts data.

## Opening a database

Use `kv.New(path, opts...)` to open or create a database.

```go
db, err := kv.New("./data", log.WithSyncEveryN(100))
```

`kv.New` passes log options through to the log layer, so the same options apply when Kival opens existing segments or creates a new database.

### Options

- `log.WithSyncStrategy(log.Always)`:
  - sync after every write
  - default behavior
- `log.WithSyncStrategy(log.EveryN)`:
  - sync after every `N` writes
- `log.WithSyncEveryN(n)`:
  - controls how many writes happen before syncing when using `EveryN`
  - default is `1`

See [`log.New`](../log/log.go) and [`log.Open`](../log/log.go) for the option flow.

## Writing data

`Put(key, value)` appends the record to the active log.

The key is also stored in the in-memory index so later reads can find the segment and offset quickly.

## Log rotation

Rotation happens when appending a record would exceed `MaxDataFileSize`.

The flow is:

1. Kival estimates the encoded record size.
2. If the current segment would go past the size limit, `Append` returns `ErrCapacityExceeded`.
3. `kv.Put` catches that error and creates a new active log.
4. The old log becomes read-only and stays available for reads until compaction.

Relevant code:

- [`haveExceededCapacity`](../log/log.go)
- [`Append`](../log/log.go)
- [`rotateActiveLog`](../kv/kv.go)

`MaxDataFileSize` is intentionally small in this project so tests can exercise rotation quickly.

## Reading data

`Get(key)` resolves the key through the in-memory index and reads from the correct log segment.

If the key is not present, Kival returns `ErrKeyNotFound`.

## Deletions

`Del(key)` writes a tombstone record and removes the key from the index.

That tombstone is important during recovery because it prevents older values from being resurrected when the index is rebuilt.

## Compaction

Compaction is manual, not automatic.

Call `Merge()` when you want to rewrite live keys into a fresh log and remove stale segments.

What `Merge()` does:

1. creates a new log file
2. rewrites every currently live key/value pair
3. marks old segments read-only
4. closes and removes the old `.data` files
5. resets the old segment map

Relevant code: [`Merge`](../kv/kv.go)

## Important notes

- `MaxDataFileSize` is intentionally small for testing and learning.
- Rotation happens on write.
- Compaction happens only when you call `Merge()`.
- Keys and values are stored as `[]byte`.
