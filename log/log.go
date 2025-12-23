package log

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	HeaderSize = 8
)

type Log interface {
	Append(key, val []byte) (pos LogPosition, err error)
	Read(offset int64) ([]byte, error)
	ReadAt(pos LogPosition) ([]byte, error)
	Size() int64
	ID() uint32
}

// LogPosition
//
//	key -> LogPosition
type LogPosition struct {
	FileID uint32 // which segment file
	Offset int64  // where the record starts inside that file
}

type logFile struct {
	id       uint32
	file     *os.File
	writePos int64 // where the next Write should happen
}

func BuildIndex(lf *logFile) (map[string]LogPosition, error) {
	idx := make(map[string]LogPosition)
	offset := int64(0)
	f := lf.file

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()

	for offset < size {
		header := make([]byte, HeaderSize)
		_, err := f.ReadAt(header, offset)
		if err != nil {
			return nil, err
		}

		keyLen := binary.LittleEndian.Uint32(header[0:4])
		valLen := binary.LittleEndian.Uint32(header[4:8])

		entryStart := offset
		offset += HeaderSize

		key := make([]byte, keyLen)
		_, err = f.ReadAt(key, offset)
		if err != nil {
			return nil, err
		}
		offset += int64(keyLen)

		// We don't need to read the value into memory now
		offset += int64(valLen)

		idx[string(key)] = LogPosition{
			FileID: lf.id,
			Offset: entryStart,
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

	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[0:4], uint32(len(key)))
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(val)))

	offset := start

	n, err := d.file.WriteAt(header, offset)
	if err != nil {
		return LogPosition{}, nil
	}

	offset += int64(n)
	n, err = d.file.WriteAt(key, offset)
	if err != nil {
		return LogPosition{}, nil
	}

	offset += int64(n)
	n, err = d.file.WriteAt(val, offset)
	if err != nil {
		return LogPosition{}, nil
	}

	d.writePos = offset + int64(n)

	return LogPosition{
		FileID: d.id,
		Offset: start,
	}, nil
}

func (d *logFile) Read(offset int64) ([]byte, error) {
	return []byte{}, nil
}

func (d *logFile) ReadAt(pos LogPosition) ([]byte, error) {
	offset := pos.Offset

	header := make([]byte, 8)
	if _, err := d.file.ReadAt(header, offset); err != nil {
		return nil, err
	}

	keyLen := binary.LittleEndian.Uint32(header[0:4])
	valueLen := binary.LittleEndian.Uint32(header[4:8])

	// skip the key — for Get we only want the value
	offset += 8 + int64(keyLen)

	val := make([]byte, valueLen)
	if _, err := d.file.ReadAt(val, offset); err != nil {
		return nil, err
	}

	return val, nil
}

func (d *logFile) Size() int64 {
	return 0
}

func (d *logFile) ID() uint32 {
	return d.id
}
