package kv

import (
	"errors"
	"time"

	"github.com/1garo/kival/log"
	"github.com/1garo/kival/record"
)

var (
	ErrNotFound = errors.New("key not found in db")
)

type KV interface {
	Set(key string, data []byte) error
	Get(key string) ([]byte, error)
	Del(key string)
}

type kv struct {
	activeLog log.Log
	index     map[string]log.LogPosition
	logs      map[uint32]log.Log
}

func OpenStore(dir string) *kv {
	// 1: scan dir for segment files
	// 2: extract IDs
	// 3: sort IDs
	// 4: for each ID → NewLogFile()
	// 5: activeLog = highest-ID file
	// 6: rebuild index by scanning logs
	return &kv{}
}

var _ KV = (*kv)(nil)

func New(l log.Log) kv {
	return kv{
		activeLog: nil,
		index:     map[string]log.LogPosition{},
		logs:      map[uint32]log.Log{},
	}
}

func (m kv) Set(key string, data []byte) error {
	rec := record.Record{
		Key:       []byte(key),
		Value:     data,
		Timestamp: time.Now().Unix(),
	}
	offset, err := m.activeLog.Append(record.Encode(rec))
	if err != nil {
		return errors.New("cannot append encoded data into db")
	}

	pos := log.LogPosition{
		FileID: m.activeLog.ID(),
		Offset: offset,
	}

	m.index[key] = pos
	return nil
}

func (m kv) Get(key string) ([]byte, error) {
	pos, ok := m.index[key]
	if !ok {
		return nil, ErrNotFound
	}

	raw, err := m.logs[pos.FileID].Read(pos.Offset)
	if err != nil {
		return nil, errors.New("failed to read offset from file")
	}

	rec := record.Decode(raw)

	return rec.Value, nil
}

func (m kv) Del(key string) {}
