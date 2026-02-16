package kv

import (
	"errors"
	"fmt"

	"github.com/1garo/kival/log"
)

const DEFAULT_DB_PATH = "./data"

var (
	ErrKeyNotFound = errors.New("key not found in db")
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
	activeLog, logs, index, err := log.Open(path)
	if err != nil {
		return nil, err
	}

	l := make(map[uint32]log.Log, len(logs))
	for id, lf := range logs {
		l[id] = lf
	}
	return &kv{
		activeLog: activeLog,
		keyDir:    index,
		logs:      l,
	}, nil
}

var _ KV = (*kv)(nil)

// rotateActiveLog rotates the active log file and creates a new one.
func (m *kv) rotateActiveLog(key, data []byte) (log.LogPosition, error) {
	m.activeLog.MarkReadOnly()

	newLog, err := log.New(m.activeLog.ID()+1, DEFAULT_DB_PATH)
	if err != nil {
		return log.LogPosition{}, fmt.Errorf("cannot create a new after capacity exceeded: %v", err)
	}
	m.logs[m.activeLog.ID()] = m.activeLog

	m.activeLog = newLog
	pos, err := m.activeLog.Append(key, data)
	if err != nil {
		return log.LogPosition{}, fmt.Errorf("failed to append to rotated log: %v", err)
	}

	return pos, nil
}

func (m *kv) Put(key []byte, data []byte) error {
	pos, err := m.activeLog.Append(key, data)
	if err != nil {
		if errors.Is(err, log.ErrCapacityExceeded) {
			fmt.Println("entrou aqui?")
			p, err := m.rotateActiveLog(key, data)
			if err != nil {
				return err
			}
			pos = p
		} else {
			return fmt.Errorf("cannot append encoded data into db: %v", err)
		}
	}

	m.keyDir[string(key)] = pos
	return nil
}

func (m *kv) Get(key []byte) ([]byte, error) {
	pos, ok := m.keyDir[string(key)]
	if !ok {
		return nil, ErrKeyNotFound
	}

	if active, ok := m.logs[pos.FileID]; ok {
		return active.ReadAt(pos)
	}
	return m.activeLog.ReadAt(pos)
}

func (m *kv) Del(key []byte) error {
	if _, ok := m.keyDir[string(key)]; !ok {
		return ErrKeyNotFound
	}

	if _, err := m.activeLog.Append(key, nil); err != nil {
		return fmt.Errorf("cannot append encoded data into db: %w", err)
	}

	delete(m.keyDir, string(key))
	return nil
}
