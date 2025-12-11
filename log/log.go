package log

import (
	"fmt"
	"os"
	"path/filepath"
)

type Log interface {
	Append(record []byte) (offset int64, err error)
	Read(offset int64) ([]byte, error)
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
	id   uint32
	file *os.File
	size int64
}

func New(id uint32, dir string) (*logFile, error) {
	path := logFilename(id, dir)

	f, err := os.OpenFile(path,
		os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}


	return &logFile{
		id:   id,
		file: f,
		size: stat.Size(),
	}, nil
}

func logFilename(id uint32, dir string) string {
	return filepath.Join(dir, fmt.Sprintf("%08d.data", id))
}

func (d logFile) Append(record []byte) (int64, error) {
	return 0, nil
}

func (d logFile) Read(offset int64) ([]byte, error) {
	return []byte{}, nil
}

func (d logFile) Size() int64 {
	return 0
}

func (d logFile) ID() uint32 {
	return d.id
}
