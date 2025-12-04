package kv

import "errors"

var (
	ErrKeyAlreadyPresent = errors.New("key already present into the database")
	ErrKeyNotFound       = errors.New("key not found in the database")
)

type KV interface {
	Set(key string, data []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

type Iterator interface {
   HasNext() bool
   Next() (key []byte, val []byte)
}

type MemDB struct {
	data map[string][]byte
}

func New() MemDB {
	return MemDB{
		data: make(map[string][]byte), 
	}
}

var _ KV = (*MemDB)(nil)

func (m MemDB) Set(key string, data []byte) error {
	if _, ok := m.data[key]; ok {
		return ErrKeyAlreadyPresent
	}

	m.data[key] = data
	return nil
}

func (m MemDB) Get(key string) ([]byte, error) {
	b, ok := m.data[key]
	if !ok {
		return []byte{}, ErrKeyNotFound
	}

	return b, nil
}

func (m MemDB) Delete(key string) error {
	if _, ok := m.data[key]; !ok {
		return ErrKeyNotFound
	}

	delete(m.data, key)
	return nil
}
