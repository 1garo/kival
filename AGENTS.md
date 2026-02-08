# AGENTS.md - Guidelines for Coding Agents

This document provides guidelines for agentic coding agents working in the Kival codebase. Kival is a Bitcask-inspired key-value store implemented in Go.

## Project Overview

- **Purpose**: Learning-focused, append-only storage engine with in-memory indexing
- **Architecture**: Log-structured storage with crash recovery
- **Packages**:
  - `log`: Append-only log file management with rotation
  - `record`: Binary record encoding/decoding with CRC validation
  - `kv`: Key-value store API layer
- **Key Constants**: `MaxDataFileSize = 500` (bytes for testing)

## Build/Lint/Test Commands

### Testing Commands
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./log
go test ./record
go test ./kv

# Run single test
go test ./log -v -run TestSpecificFunctionName
go test ./log -run TestLog_ReadAt_ReturnsAppendedValue

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

### Build Commands
```bash
# Build main application
go build -o kival ./main.go

# Build with verbose output
go build -v -o kival ./main.go

# Format code
go fmt ./...

# Run linter (if available)
golint ./...
go vet ./...
```

## Code Style Guidelines

### Import Organization
Follow Uber Go Style Guide order:
1. Standard library imports
2. Third-party imports  
3. Project imports

```go
import (
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "testing"
    "time"

    "github.com/1garo/kival/log"
    "github.com/1garo/kival/record"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)
```

### Error Handling
- Use `errors.New()` for static error strings
- Use `fmt.Errorf()` for dynamic errors with context
- Prefix error variables with `Err` for exported, `err` for unexported
- Handle errors once, don't log and return
- Use `errors.Is()` and `errors.As()` for error matching

```go
var (
    ErrNotFound = errors.New("key not found in db")
    errInvalidInput = errors.New("invalid input parameters")
)

// Good error handling
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Error matching
if errors.Is(err, log.ErrCapacityExceeded) {
    // handle capacity exceeded
}
```

### Naming Conventions
- **Interfaces**: Simple names, often single words (e.g., `Log`, `KV`)
- **Types**: CamelCase for exported, camelCase for unexported
- **Functions**: CamelCase, verbs for actions
- **Variables**: camelCase, descriptive names
- **Constants**: UPPER_SNAKE_CASE or PascalCase for exported
- **Files**: lowercase, single word per package

```go
type Log interface { ... }  // Interface
type logFile struct { ... }  // Unexported struct
func New(id uint32, dir string) (*logFile, error) { ... }  // Constructor
const MaxDataFileSize = 500  // Exported constant
var errClosed = errors.New("log is closed")  // Unexported error
```

### Testing Patterns
- Use table-driven tests for multiple scenarios
- Use `t.Helper()` for helper functions
- Use `t.Cleanup()` for resource cleanup
- Use `require.NoError()` for setup, `assert.Equal()` for verification
- Follow naming: `TestPackage_Function_Scenario`

```go
func TestLog_Append_CapacityExceeded(t *testing.T) {
    l := newTestLog(t)  // Helper creates log with cleanup
    
    key := []byte("test")
    value := []byte("value")
    
    _, err := l.Append(key, value)
    assert.ErrorIs(t, err, log.ErrCapacityExceeded)
}

func newTestLog(t *testing.T) log.Log {
    t.Helper()
    dir := t.TempDir()
    l, err := log.New(1, dir)
    require.NoError(t, err)
    t.Cleanup(func() { _ = l.Close() })
    return l
}
```

### Interface Compliance
- Add compile-time interface checks
- Use pointer receivers for methods that modify state
- Use value receivers for methods that don't modify state

```go
var _ Log = (*logFile)(nil)  // Interface compliance check

func (d *logFile) Append(key, val []byte) (LogPosition, error) { ... }  // Pointer receiver
func (d logFile) ID() uint32 { ... }  // Value receiver
```

### Constants and Magic Numbers
- Use named constants instead of magic numbers
- Group related constants
- Add comments explaining purpose

```go
const (
    HeaderSize  = uint32(16) // crc(4) + timestamp(4) + keySize(4) + valSize(4)
    CustomEpoch = 1704067200 // first commit to project - 2025-12-04 UTC
)
```

### File I/O and Resource Management
- Use `defer` for cleanup (Close, Sync, etc.)
- Use `os.O_EXCL` for atomic file creation
- Use temporary files with rename pattern
- Handle all file operation errors

### Struct Design
- Export fields only when necessary
- Use field tags for marshaled structs
- Prefer composition over embedding
- Keep structs small and focused

## Package-Specific Guidelines

### Log Package
- Manages append-only files with rotation at `MaxDataFileSize`
- Uses `LogPosition{FileID, ValuePos, ValueSize, timestamp}` for addressing
- Key errors: `ErrCapacityExceeded`, `ErrReadOnlySegment`, `ErrLogClosed`
- Key methods: `Append()`, `ReadAt()`, `Size()`, `ID()`, `MarkReadOnly()`

### Record Package  
- Binary format: `CRC(4) + Timestamp(4) + KeySize(4) + ValueSize(4) + Key + Value`
- Uses CRC32 for corruption detection
- Key errors: `ErrEmptyKey`, `ErrPartialWrite`, `ErrCorruptRecord`, `ErrEncodeInput`
- Key functions: `Encode()`, `Decode()`, `GenerateCRC()`

### KV Package
- Implements key-value store interface over log package
- Handles log rotation when capacity exceeded
- Uses tombstone records for deletions
- Key interface: `KV` with `Put()`, `Get()`, `Del()` methods

## Development Workflow

1. **Write Tests First**: Create failing tests for new functionality
2. **Implementation**: Write minimal code to make tests pass
3. **Refactor**: Clean up while maintaining test coverage
4. **Verify**: Run `go test -race -cover ./...`
5. **Format**: Run `go fmt ./...` before committing

## Performance Considerations

- Tests use `MaxDataFileSize = 500` bytes for fast execution
- Use `t.TempDir()` for test isolation and cleanup
- Avoid unnecessary allocations in hot paths
- Use `sync.RWMutex` for concurrent access protection

## Common Pitfalls to Avoid

- Don't use `init()` functions in production code
- Don't use `panic()` in production code (use errors instead)
- Don't ignore returned errors
- Don't export unneeded fields or functions
- Don't use `log.Fatal()` outside of `main()`
- Don't create goroutines without proper lifecycle management