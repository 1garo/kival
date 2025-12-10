package kv

type KV interface {
	Set(key []byte, data []byte) 
	Get(key []byte) ([]byte, error)
	Del(key []byte) 
}

type Iterator interface {
	HasNext() bool
	Next() (key []byte, val []byte)
}

type DB struct {}

func New() DB {
	return DB{}
}

var _ KV = (*DB)(nil)

func (m DB) Set(key []byte, data []byte) {}

func (m DB) Get(key []byte) ([]byte, error) {
	return []byte{}, nil
}

func (m DB) Del(key []byte) {}
