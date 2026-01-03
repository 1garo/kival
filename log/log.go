package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/1garo/kival/record"
)

var (
	ErrCapacityExceeded = errors.New("capacity exceeded creation failed")
)

// const MaxDataFileSize = 128 * 1024 * 1024 // 128 MB
const MaxDataFileSize = 500 // 500 Bytes

type Log interface {
	Append(key, val []byte) (pos LogPosition, err error)
	ReadAt(pos LogPosition) ([]byte, error)
	Size() int64
	ID() uint32
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

func Open(path string) (*logFile, map[string]LogPosition, error) {
	// 1. ensure directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, map[string]LogPosition{}, err
	}

	// 2. open active log file
	lf, err := New(1, path) // we’ll improve file ID later
	if err != nil {
		return nil, map[string]LogPosition{}, err
	}

	// 3. build index by scanning
	index, err := BuildIndex(lf)
	if err != nil {
		return nil, map[string]LogPosition{}, err
	}

	return lf, index, err
}

type logFile struct {
	id       uint32
	file     *os.File
	writePos int64 // where the next Write should happen
}

func BuildIndex(lf *logFile) (map[string]LogPosition, error) {
	idx := make(map[string]LogPosition)
	offset := int64(0)

	stat, err := lf.file.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()

	for offset < size {
		start := offset
		rec, bytesRead, err := record.Decode(lf.file, offset)
		if err != nil {
			if err == io.EOF || errors.Is(err, record.ErrPartialWrite) || errors.Is(err, record.ErrCorruptRecord) {
				break // stop reading this file
			}
			return nil, err
		}

		offset += bytesRead

		isTombstone := rec.ValueSize == 0
		if isTombstone {
			delete(idx, string(rec.Key))
			continue
		}

		idx[string(rec.Key)] = LogPosition{
			FileID:    lf.id,
			ValuePos:  start,
			ValueSize: rec.ValueSize,
			timestamp: rec.Timestamp,
		}
	}

	// update WritePos to end of file
	lf.writePos = offset

	return idx, nil
}

func New(id uint32, dir string) (*logFile, error) {
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

//func logFilename(id uint32, dir string) string {
//	return filepath.Join(dir, fmt.Sprintf("%08d.data", id))
//}

func (d *logFile) Append(key, val []byte) (LogPosition, error) {
	start := d.writePos

	keySize := uint32(len(key))
	valSize := uint32(len(val))
	recordSize := record.HeaderSize + keySize + valSize
	// TODO: probably this needs to become a method -> ensureCapacity()
	exceedCapacity := int64(recordSize)+start > MaxDataFileSize
	if exceedCapacity {
		err := d.file.Close()
		if err != nil {
			return LogPosition{}, fmt.Errorf("%w: %v", ErrCapacityExceeded, err)
		}

		fd, err := New(d.id+1, "./data")
		d.file = fd.file
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
