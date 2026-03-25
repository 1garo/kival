// Package kv is the entry for the database
package kv

import (
	"errors"
	"fmt"
	"os"

	"github.com/1garo/kival/log"
)

const DefaultDBPath = "./data"

var ErrKeyNotFound = errors.New("key not found in db")

type KV interface {
	Put(key []byte, data []byte) error
	Get(key []byte) ([]byte, error)
	Del(key []byte) error
	Merge() error
}

type kv struct {
	activeLog log.Log
	keyDir    map[string]log.LogPosition
	logs      map[uint32]log.Log
	dbPath    string
}

// New creates a new database or sync based on data into path
func New(path string, opts ...log.Option) (*kv, error) {
	activeLog, logs, index, err := log.Open(path, opts...)
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
		dbPath:    path,
	}, nil
}

var _ KV = (*kv)(nil)

// rotateActiveLog rotates the active log file, appends data, and returns the position.
func (m *kv) rotateActiveLog(key, data []byte) (log.LogPosition, error) {
	m.activeLog.MarkReadOnly()
	m.logs[m.activeLog.ID()] = m.activeLog

	newLog, err := log.New(m.activeLog.ID()+1, m.dbPath)
	if err != nil {
		return log.LogPosition{}, fmt.Errorf("cannot create new log: %w", err)
	}

	m.activeLog = newLog

	pos, err := newLog.Append(key, data)
	if err != nil {
		return log.LogPosition{}, fmt.Errorf("failed to append to rotated log: %w", err)
	}

	return pos, nil
}

// Put add a new key and value to the active log
func (m *kv) Put(key []byte, data []byte) error {
	pos, err := m.activeLog.Append(key, data)
	if err != nil {
		if errors.Is(err, log.ErrCapacityExceeded) {
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

// Get get a value from the log based on the key
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

// Del a key from the active log
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

// Merge merges all the logs in the db into a single log file
func (m *kv) Merge() error {
	if len(m.logs) == 0 {
		return nil
	}

	var compactedLog log.Log
	var err error
	compactedLog, err = log.New(m.activeLog.ID()+1, m.dbPath)
	if err != nil {
		return fmt.Errorf("cannot create new compacted log: %w", err)
	}

	for key := range m.keyDir {
		val, err := m.Get([]byte(key))
		if err != nil {
			return fmt.Errorf("failed to get value: %w", err)
		}

		pos, err := compactedLog.Append([]byte(key), val)
		if err != nil {
			if errors.Is(err, log.ErrCapacityExceeded) {
				pos, err = m.rotateActiveLog([]byte(key), val)
				if err != nil {
					return err
				}
				compactedLog = m.activeLog
			} else {
				return fmt.Errorf("failed to append: %w", err)
			}
		}

		m.keyDir[key] = pos
	}

	for _, l := range m.logs {
		l.MarkReadOnly()
	}

	m.activeLog = compactedLog

	for id, l := range m.logs {
		_ = l.Close()
		filename := fmt.Sprintf("%s/%d.data", m.dbPath, id)
		_ = os.Remove(filename)
	}

	m.logs = make(map[uint32]log.Log)

	return nil
}
