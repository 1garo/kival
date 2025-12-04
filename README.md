# Go Key-Value Database — Milestone Roadmap

This document outlines clear milestones, each with deliverables and **small, illustrative code snippets**—not full implementations. Every milestone is scoped to be achievable in **under ~2 hours** of focused work.

---

## **Milestone 1 — Define Scope & Minimal API**

### **Goals**

* Establish what the DB will and will *not* support.
* Define the essential public API.

### **Deliverables**

* `README` section describing system behavior.
* API interface with no logic.

### **Snippet**

```go
// DB defines the basic key-value operations
// No persistence yet
interface DB {
    Set(key string, value []byte) error
    Get(key string) ([]byte, error)
    Delete(key string) error
}
```

---

## **Milestone 2 — In-Memory Prototype**

### **Goals**

* Create a working version based entirely on a Go map.

### **Deliverables**

* `memdb.go` containing a struct and method stubs.
* A few basic tests.

### **Snippet**

```go
type MemDB struct {
    data map[string][]byte
}

func NewMemDB() *MemDB {
    return &MemDB{data: make(map[string][]byte)}
}
```

---

## **Milestone 3 — Define On-Disk Record Format**

### **Goals**

* Document binary record structure.
* Decide how tombstones are encoded.

### **Deliverables**

* `record_format.md` with structure like:

  * keySize (uint32)
  * valueSize (uint32)
  * tombstone (byte)
  * key bytes
  * value bytes

### **Snippet**

```go
// Example struct used only for documentation purposes
// Not necessarily used directly
struct Record {
    KeySize   uint32
    ValueSize uint32
    Tombstone uint8 // 0 = alive, 1 = deleted
    Key       []byte
    Value     []byte
}
```

---

## **Milestone 4 — Append-Only Writer**

### **Goals**

* Implement the ability to append encoded records to a file.
* No reading logic yet.

### **Deliverables**

* `writer.go` with `appendRecord(record)` function.

### **Snippet**

```go
func (db *BitcaskDB) append(key, value []byte, tombstone bool) (int64, error) {
    // Encode sizes
    buf := new(bytes.Buffer)
    binary.Write(buf, binary.LittleEndian, uint32(len(key)))
    binary.Write(buf, binary.LittleEndian, uint32(len(value)))
    if tombstone {
        buf.WriteByte(1)
    } else {
        buf.WriteByte(0)
    }
    buf.Write(key)
    buf.Write(value)

    // Append
    offset, _ := db.file.Seek(0, io.SeekEnd)
    _, err := db.file.Write(buf.Bytes())
    return offset, err
}
```

---

## **Milestone 5 — Startup Index Rebuild**

### **Goals**

* Scan the entire log file and rebuild the in-memory index (key → offset).

### **Deliverables**

* `rebuild.go` with sequential scan logic.

### **Snippet**

```go
// Pseudocode — not full implementation
for {
    rec, offset, err := readNextRecord(file)
    if err == io.EOF {
        break
    }
    index[string(rec.Key)] = offset
}
```

---

## **Milestone 6 — Support Deletions (Tombstones)**

### **Goals**

* Write tombstone records to file.
* Update index rebuild logic to ignore deleted keys.

### **Deliverables**

* `Delete()` implementation using append.

### **Snippet**

```go
func (db *BitcaskDB) Delete(key string) error {
    _, err := db.append([]byte(key), nil, true)
    delete(db.index, key)
    return err
}
```

---

## **Milestone 7 — Compaction (Basic)**

### **Goals**

* Merge the log by scanning keys and rewriting only latest versions.

### **Deliverables**

* `compact.go` implementing a simple compaction command.

### **Snippet**

```go
for key, offset := range db.index {
    rec := readAtOffset(offset)
    newFile.Write(rec.encode())
}
```

---

## **Milestone 8 — Concurrency (Minimal)**

### **Goals**

* Add a mutex protecting writes & reads.

### **Deliverables**

* `RWMutex` embedded in the DB struct.

### **Snippet**

```go
type BitcaskDB struct {
    mu    sync.RWMutex
    index map[string]int64
    file  *os.File
}
```

---

## **Milestone 9 — Testing & Benchmarks**

### **Goals**

* Write unit tests for Get/Set/Delete.
* Write lightweight benchmarks.

### **Deliverables**

* `_test.go` files
* Basic benchmark output.

### **Snippet**

```go
func BenchmarkSet(b *testing.B) {
    db := NewMemDB()
    for i := 0; i < b.N; i++ {
        db.Set("key", []byte("value"))
    }
}
```

---

## **Milestone 10 — Optional Stretch Features**

### **Examples**

* TTL
* WAL
* Multi-file segments
* Bloom filters

### **Deliverables**

* Only design documents unless you choose to implement them.

---

If you'd like, I can now:

* add checkboxes,
* add a timeline,
* add references, or
* turn milestones into GitHub issues. 🚀
# kival
