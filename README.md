# Kival - A Bitcask-Inspired Key-Value Store (Go)

A learning-focused, Bitcask-inspired key-value store implemented in Go.

This project explores how log-structured storage engines work under the hood: append-only logs, in-memory indexes, crash recovery, and data integrity.

The goal is **correctness first**, then **performance**, while keeping the implementation small, inspectable, and educational.

## Example

There is a small runnable example in [`example/`](./example) that shows the basic API in action:

- open a database
- write a key/value pair
- read it back
- delete it
- write it again

Run it from the example module directory:

```bash
cd example
go run .
```

The example uses the published `github.com/1garo/kival` module, so it is a good reference for external consumers of the package.

## How It Works

For a short explanation of Kival's storage model, log rotation, and compaction, see [docs/how-it-works.md](./docs/how-it-works.md).
