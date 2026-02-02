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

var (
	ErrCapacityExceeded = errors.New("capacity exceeded creation failed")
	ErrReadOnlySegment  = errors.New("file is in readonly state, cannot write to it")
	ErrLogClosed        = errors.New("log is closed")
)

// const MaxDataFileSize = 128 * 1024 * 1024 // 128 MB
const MaxDataFileSize = 500 // 500 Bytes

type Log interface {
	Append(key, val []byte) (pos LogPosition, err error)
	ReadAt(pos LogPosition) ([]byte, error)
	Size() int64
	ID() uint32
	Close() error
	MarkReadOnly()
}

// LogPosition
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

func Open(path string) (*logFile, map[uint32]*logFile, map[string]LogPosition, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, nil, nil, err
	}

	files, _ := filepath.Glob(filepath.Join(path, "*.data"))
	sort.Strings(files)

	index := make(map[string]LogPosition)
	logs := make(map[uint32]*logFile)

	if len(files) == 0 {
		lf, err := New(1, path)
		if err != nil {
			return nil, nil, nil, err
		}
		return lf, logs, index, nil
	}

	var active *logFile

	for i, f := range files {
		id := parseFileID(f)

		lf, err := New(id, path)
		if err != nil {
			return nil, nil, nil, err
		}

		if err := lf.BuildIndex(index); err != nil {
			return nil, nil, nil, err
		}

		if i == len(files)-1 {
			active = lf
		} else {
			logs[id] = lf
		}
	}

	return active, logs, index, nil
}

type logFile struct {
	id       uint32
	file     *os.File
	writePos int64 // where the next Write should happen
	readOnly bool
	closed   bool
}

func (d *logFile) BuildIndex(idx map[string]LogPosition) error {
	offset := int64(0)

	stat, err := d.file.Stat()
	if err != nil {
		return err
	}
	size := stat.Size()

	for offset < size {
		start := offset
		rec, bytesRead, err := record.Decode(d.file, offset)
		if err != nil {
			if err == io.EOF || errors.Is(err, record.ErrPartialWrite) || errors.Is(err, record.ErrCorruptRecord) {
				break // stop reading this file
			}
			return err
		}

		offset += bytesRead

		isTombstone := rec.ValueSize == 0
		if isTombstone {
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

func New(id uint32, dir string) (*logFile, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(
		filepath.Join(dir, fmt.Sprintf("%d.data", id)),
		os.O_CREATE|os.O_RDWR,
		0644,
	)
	if err != nil {
		return nil, err
	}

	// Seek to end — Bitcask always appends.
	pos, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	return &logFile{
		file:     f,
		id:       id,
		writePos: pos,
	}, nil
}

func (d *logFile) Append(key, val []byte) (LogPosition, error) {
	if d.readOnly {
		return LogPosition{}, ErrReadOnlySegment
	}
	start := d.writePos

	keySize := uint32(len(key))
	valSize := uint32(len(val))
	recordSize := record.HeaderSize + keySize + valSize
	// TODO: probably this needs to become a method -> ensureCapacity()
	exceedCapacity := int64(recordSize)+start > MaxDataFileSize
	if exceedCapacity {
		return LogPosition{}, ErrCapacityExceeded
	}
	buf := record.Encode(key, val)

	// add saveData here
	n, err := d.file.WriteAt(buf, start)
	if err != nil {
		return LogPosition{}, err
	}

	if err = d.file.Sync(); err != nil {
		return LogPosition{}, err
	}

	d.writePos = start + int64(n)

	return NewLogPosition(
		d.id,
		uint32(len(val)),
		uint32(time.Now().Unix()),
		start,
	), nil
}

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

func (d *logFile) Size() int64 {
	stat, _ := d.file.Stat()
	return stat.Size()
}

func (d *logFile) ID() uint32 {
	return d.id
}

func (d *logFile) Close() error {
	d.closed = true
	return d.file.Close()
}

func (d *logFile) MarkReadOnly() {
	d.readOnly = true
}
