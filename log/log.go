package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/1garo/kival/record"
)

type (
	Logs  map[uint32]*logFile
	Index map[string]LogPosition
)

type SyncStrategy int

const (
	Always SyncStrategy = iota
	EveryN
)

var (
	ErrCapacityExceeded = errors.New("capacity exceeded creation failed")
	ErrReadOnlySegment  = errors.New("file is in readonly state, cannot write to it")
	ErrLogClosed        = errors.New("log is closed")
)

var MaxDataFileSize = 1500 // 1.5 KB for faster tests

type Log interface {
	Append(key, val []byte) (pos LogPosition, err error)
	ReadAt(pos LogPosition) ([]byte, error)
	Size() int64
	ID() uint32
	Close() error
	MarkReadOnly()
	WriteCount() int32
}

// LogPosition is the position of the data inside the log files
type LogPosition struct {
	FileID    uint32 // which segment file
	ValuePos  int64  // where the record starts inside that file
	ValueSize uint32
	timestamp uint32
}

func NewLogPosition(fileID, valueSize, timestamp uint32, valuePos int64) LogPosition {
	return LogPosition{
		FileID:    fileID,
		ValuePos:  valuePos,
		ValueSize: valueSize,
		timestamp: timestamp,
	}
}

func parseFileID(name string) uint32 {
	base := filepath.Base(name)
	idStr := strings.TrimSuffix(base, ".data")
	id, _ := strconv.ParseUint(idStr, 10, 32)
	return uint32(id)
}

// Option type is to configure your log
type Option func(*logFile) error

// WithSyncStrategy set the sync strategy to the log
func WithSyncStrategy(s SyncStrategy) Option {
	return func(lf *logFile) error {
		lf.syncStrategy = s
		return nil
	}
}

// WithSyncEveryN sync log every N writes
func WithSyncEveryN(n int32) Option {
	return func(lf *logFile) error {
		lf.syncEveryN = n
		return nil
	}
}

// Open recreates the log state from the given path.
// It goes through all the log files under the given path.
// It returns the active log file, a map of log files, a map of log positions, and an error.
func Open(path string, options ...Option) (*logFile, Logs, Index, error) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, nil, nil, err
	}

	files, _ := filepath.Glob(filepath.Join(path, "*.data"))
	sort.Slice(files, func(i, j int) bool {
		idI := parseFileID(files[i])
		idJ := parseFileID(files[j])
		return idI < idJ
	})

	index := make(Index)
	logs := make(Logs)

	if len(files) == 0 {
		lf, err := New(1, path, options...)
		if err != nil {
			return nil, nil, nil, err
		}
		return lf, logs, index, nil
	}

	var active *logFile

	for i, f := range files {
		id := parseFileID(f)

		lf, err := openExisting(id, path, options...)
		if err != nil {
			return nil, nil, nil, err
		}

		if err := lf.BuildIndex(index); err != nil {
			return nil, nil, nil, err
		}

		isLatest := i == len(files)-1
		if isLatest {
			active = lf
		} else {
			logs[id] = lf
		}
	}

	return active, logs, index, nil
}

// logFile represents a log file.
type logFile struct {
	id           uint32
	file         *os.File
	writePos     int64
	writeCount   int32
	readOnly     bool
	closed       bool
	syncStrategy SyncStrategy
	syncEveryN   int32
}

// BuildIndex builds an index of keys and their positions in the log file.
func (d *logFile) BuildIndex(idx map[string]LogPosition) error {
	offset := int64(0)

	stat, err := d.file.Stat()
	if err != nil {
		return err
	}
	fileSize := stat.Size()

	for offset < fileSize {
		start := offset
		rec, bytesRead, err := record.Decode(d.file, offset)
		if err != nil {
			if err == io.EOF || errors.Is(err, record.ErrPartialWrite) || errors.Is(err, record.ErrCorruptRecord) {
				break
			}

			return err
		}

		offset += bytesRead

		isTombstoneRecord := rec.ValueSize == 0
		if isTombstoneRecord {
			delete(idx, string(rec.Key))
			continue
		}

		idx[string(rec.Key)] = LogPosition{
			FileID:    d.id,
			ValuePos:  start,
			ValueSize: rec.ValueSize,
			timestamp: rec.Timestamp,
		}
	}

	// update WritePos to end of file
	d.writePos = offset
	return nil
}

// New creates a new log file
func New(id uint32, dir string, options ...Option) (*logFile, error) {
	l := &logFile{
		id:           id,
		syncStrategy: Always,
		syncEveryN:   1,
	}

	for _, opt := range options {
		if err := opt(l); err != nil {
			return nil, err
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	fileName := filepath.Join(dir, fmt.Sprintf("%d.data", id))
	f, err := os.OpenFile(
		fileName,
		os.O_CREATE|os.O_RDWR|os.O_TRUNC,
		0o644,
	)
	if err != nil {
		return nil, err
	}

	pos, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	l.writePos = pos
	l.file = f

	return l, nil
}

// openExisting opens an existing log file without truncating it.
func openExisting(id uint32, dir string, options ...Option) (*logFile, error) {
	l := &logFile{
		id:           id,
		syncStrategy: Always,
		syncEveryN:   1,
	}

	for _, opt := range options {
		if err := opt(l); err != nil {
			return nil, err
		}
	}

	fileName := filepath.Join(dir, fmt.Sprintf("%d.data", id))
	f, err := os.OpenFile(
		fileName,
		os.O_RDWR,
		0o644,
	)
	if err != nil {
		return nil, err
	}

	pos, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	l.writePos = pos
	l.file = f

	return l, nil
}

// haveExceededCapacity checks if the log file has exceeded its capacity.
func (d *logFile) haveExceededCapacity(key, val []byte) error {
	keySize := uint32(len(key))
	valSize := uint32(len(val))

	recordSize := record.HeaderSize + keySize + valSize
	if int64(recordSize)+d.writePos > int64(MaxDataFileSize) {
		return ErrCapacityExceeded
	}
	return nil
}

// Append appends a key-value pair to the log file.
func (d *logFile) Append(key, val []byte) (LogPosition, error) {
	if d.readOnly {
		return LogPosition{}, ErrReadOnlySegment
	}
	start := d.writePos

	if err := d.haveExceededCapacity(key, val); err != nil {
		return LogPosition{}, err
	}

	buf := record.Encode(key, val)
	n, err := d.file.WriteAt(buf, d.writePos)
	if err != nil {
		return LogPosition{}, err
	}

	d.writeCount++

	switch d.syncStrategy {
	case Always:
		if err = d.file.Sync(); err != nil {
			return LogPosition{}, err
		}
	case EveryN:
		if d.writeCount == d.syncEveryN {
			if err = d.file.Sync(); err != nil {
				return LogPosition{}, err
			}

			d.writeCount = 0
		}
	}

	d.writePos += int64(n)

	return NewLogPosition(
		d.id,
		uint32(len(val)),
		uint32(time.Now().Unix()),
		start,
	), nil
}

// ReadAt reads a key-value pair from the log file at the given position.
func (d *logFile) ReadAt(pos LogPosition) ([]byte, error) {
	if d.closed {
		return nil, ErrLogClosed
	}

	rec, _, err := record.Decode(d.file, pos.ValuePos)
	if err != nil {
		return []byte{}, err
	}

	return rec.Value, nil
}

// Size return the size of the log file.
func (d *logFile) Size() int64 {
	stat, _ := d.file.Stat()
	return stat.Size()
}

// ID returns the ID of the current log file.
func (d *logFile) ID() uint32 {
	return d.id
}

// Close closes the current log file.
func (d *logFile) Close() error {
	d.closed = true
	return d.file.Close()
}

// MarkReadOnly marks the current log file as read-only.
func (d *logFile) MarkReadOnly() {
	d.readOnly = true
}

// WriteCount the amount of writes done to this file
func (d *logFile) WriteCount() int32 {
	return d.writeCount
}
