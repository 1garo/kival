package kv

import (
	"errors"
	"fmt"
	"os"

	"github.com/1garo/kival/log"
)

var (
	ErrNotFound = errors.New("key not found in db")
)

type KV interface {
	Put(key []byte, data []byte) error
	Get(key []byte) ([]byte, error)
	Del(key []byte) error
}

type kv struct {
	activeLog log.Log
	keyDir    map[string]log.LogPosition
	logs      map[uint32]log.Log
}

func Open(path string) (*kv, error) {
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
		keyDir:    index,
		logs:      map[uint32]log.Log{},
	}, nil
}

var _ KV = (*kv)(nil)

func (m kv) Put(key []byte, data []byte) error {
	pos, err := m.activeLog.Append(key, data)
	if err != nil {
		return fmt.Errorf("%w: cannot append encoded data into db", err)
	}

	m.keyDir[string(key)] = pos
	return nil
}

func (m kv) Get(key []byte) ([]byte, error) {
	pos, ok := m.keyDir[string(key)]
	if !ok {
		return nil, ErrNotFound
	}

	return m.activeLog.ReadAt(pos)
}

func (m kv) Del(key []byte) error {
	_, ok := m.keyDir[string(key)]
	if !ok {
		return ErrNotFound
	}

	_, err := m.activeLog.Append(key, []byte{})
	if err != nil {
		return fmt.Errorf("%w: cannot append encoded data into db", err)
	}

	delete(m.keyDir, string(key))
	return nil
}
