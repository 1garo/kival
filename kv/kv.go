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

// Reader reads values from a database.
type Reader interface {
	Get(key []byte) ([]byte, error)
}

// Writer writes and deletes values in a database.
type Writer interface {
	Put(key []byte, data []byte) error
	Delete(key []byte) error
}

// KV is the legacy database interface. New code should generally use *DB or
// define a smaller interface containing only the operations it needs.
type KV interface {
	Reader
	Put(key []byte, data []byte) error
	Del(key []byte) error
	Merge() error
}

// DB is an opened Kival database.
type DB struct {
	activeLog log.Log
	keyDir    map[string]log.LogPosition
	logs      map[uint32]log.Log
	dbPath    string
	closed    bool
}

// New creates a new database or sync based on data into path
func New(path string) (*DB, error) {
	activeLog, logs, index, err := log.Open(path)
	if err != nil {
		return nil, err
	}

	l := make(map[uint32]log.Log, len(logs))
	for id, lf := range logs {
		l[id] = lf
	}
	return &DB{
		activeLog: activeLog,
		keyDir:    index,
		logs:      l,
		dbPath:    path,
	}, nil
}

var _ KV = (*DB)(nil)
var _ Writer = (*DB)(nil)

// rotateActiveLog rotates the active log file, appends data, and returns the position.
func (m *DB) rotateActiveLog(key, data []byte) (log.LogPosition, error) {
	currentID := m.activeLog.ID()
	m.activeLog.MarkReadOnly()
	m.logs[currentID] = m.activeLog

	newLog, err := log.New(currentID+1, m.dbPath)
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
func (m *DB) Put(key []byte, data []byte) error {
	if m.closed {
		return log.ErrLogClosed
	}
	pos, err := m.activeLog.Append(key, data)
	if err != nil {
		if errors.Is(err, log.ErrCapacityExceeded) {
			p, err := m.rotateActiveLog(key, data)
			if err != nil {
				return err
			}
			pos = p
		} else {
			return fmt.Errorf("cannot append encoded data into db: %w", err)
		}
	}

	m.keyDir[string(key)] = pos
	return nil
}

// Get a value from the log based on the key
func (m *DB) Get(key []byte) ([]byte, error) {
	if m.closed {
		return nil, log.ErrLogClosed
	}
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
func (m *DB) Delete(key []byte) error {
	if m.closed {
		return log.ErrLogClosed
	}
	if _, ok := m.keyDir[string(key)]; !ok {
		return ErrKeyNotFound
	}

	appender, ok := m.activeLog.(log.TombstoneAppender)
	if !ok {
		return log.ErrTombstoneUnsupported
	}
	if _, err := appender.AppendTombstone(key); err != nil {
		return fmt.Errorf("cannot append encoded data into db: %w", err)
	}

	delete(m.keyDir, string(key))
	return nil
}

// Del is kept for compatibility. New code should use Delete.
func (m *DB) Del(key []byte) error {
	return m.Delete(key)
}

// Merge merges all the logs in the db into a single log file
func (m *DB) Merge() error {
	if m.closed {
		return log.ErrLogClosed
	}
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

// Close closes all files owned by the database. It is safe to call once; a
// second call returns nil so callers can safely defer it.
func (m *DB) Close() error {
	if m.closed {
		return nil
	}
	m.closed = true

	var closeErr error
	if err := m.activeLog.Close(); err != nil {
		closeErr = errors.Join(closeErr, err)
	}
	for _, l := range m.logs {
		if err := l.Close(); err != nil {
			closeErr = errors.Join(closeErr, err)
		}
	}
	return closeErr
}
