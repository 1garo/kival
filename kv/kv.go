package kv

import (
	"errors"
	"fmt"

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

func New(path string) (*kv, error) {
	activeLog, olderLogs, index, err := log.Open(path)
	if err != nil {
		return nil, err
	}

	logs := make(map[uint32]log.Log, len(olderLogs))
	for id, lf := range olderLogs {
		logs[id] = lf
	}
	return &kv{
		activeLog: activeLog,
		keyDir:    index,
		logs:      logs,
	}, nil
}

var _ KV = (*kv)(nil)

func (m *kv) Put(key []byte, data []byte) error {
	pos, err := m.activeLog.Append(key, data)
	if err != nil {
		if errors.Is(err, log.ErrCapacityExceeded) {
			m.activeLog.MarkReadOnly()

			newLog, err := log.New(m.activeLog.ID()+1, "./data")
			if err != nil {
				return fmt.Errorf("cannot create a new after capacity exceeded: %v", err)
			}
			// save for future reads on old files
			m.logs[m.activeLog.ID()] = m.activeLog

			m.activeLog = newLog
			// TODO: maybe there is a better way to do this
			pos, err = m.activeLog.Append(key, data)
			if err != nil {
				return err
			}
		}

		return fmt.Errorf("cannot append encoded data into db: %v", err)
	}

	m.keyDir[string(key)] = pos
	return nil
}

func (m *kv) Get(key []byte) ([]byte, error) {
	pos, ok := m.keyDir[string(key)]
	if !ok {
		return nil, ErrNotFound
	}

	if active, ok := m.logs[pos.FileID]; ok {
		return active.ReadAt(pos)
	}
	return m.activeLog.ReadAt(pos)
}

func (m *kv) Del(key []byte) error {
	if _, ok := m.keyDir[string(key)]; !ok {
		return ErrNotFound
	}

	// add tombstone record
	if _, err := m.activeLog.Append(key, nil); err != nil {
		return fmt.Errorf("%w: cannot append encoded data into db", err)
	}

	delete(m.keyDir, string(key))
	return nil
}
