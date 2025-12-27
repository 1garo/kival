package log

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/1garo/kival/record"
)

const (
	HeaderSize = 12
)

type Log interface {
	Append(key, val []byte) (pos LogPosition, err error)
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

		keyLen := binary.LittleEndian.Uint32(header[4:8])
		valLen := binary.LittleEndian.Uint32(header[8:12])

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

	buf := record.Encode(key, val)

	// we should us SaveData2 here instead
	n, err := d.file.WriteAt(buf, start)
	if err != nil {
		return LogPosition{}, err
	}

	d.writePos = start + int64(n)

	return LogPosition{
		FileID: d.id,
		Offset: start,
	}, nil
}

func (d *logFile) ReadAt(pos LogPosition) ([]byte, error) {
	rec, err := record.Decode(d.file, pos.Offset)
	if err != nil {
		return []byte{}, err
	}

	return rec.Value, nil
}

func (d *logFile) Size() int64 {
	return 0
}

func (d *logFile) ID() uint32 {
	return d.id
}
