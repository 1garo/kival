package kv

import (
	"errors"
	"os"

	"github.com/1garo/kival/log"
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

func OpenStore(path string) (*kv, error) {
	// 1. ensure directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}

	// 2. open active log file
	lf, err := log.New(1, path) // we’ll improve file ID later
	if err != nil {
		return nil, err
	}

	// 3. build index by scanning
	index, err := log.BuildIndex(lf)
	if err != nil {
		return nil, err
	}

	return &kv{
		activeLog: lf,
		index:     index,
		logs:      make(map[uint32]log.Log, 0),
	}, nil
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
	//rec := record.Record{
	//	Key:       []byte(key),
	//	Value:     data,
	//	Timestamp: time.Now().Unix(),
	//}
	pos, err := m.activeLog.Append([]byte(key), data)
	if err != nil {
		return errors.New("cannot append encoded data into db")
	}

	m.index[key] = pos
	return nil
}

func (m kv) Get(key string) ([]byte, error) {
	pos, ok := m.index[key]
	if !ok {
		return nil, ErrNotFound
	}

	return m.activeLog.ReadAt(pos)
}

func (m kv) Del(key string) {}
