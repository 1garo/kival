package log

type Log interface {
	Append(record []byte) (offset int64, err error)
	Read(offset int64) ([]byte, error)
	Size() int64
}

type DBLog struct{}

//	LogPosition
//		key -> LogPosition
type LogPosition struct {
    FileID uint32
    Offset int64
    Size   uint32
}
