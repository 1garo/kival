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

type MemDB struct {}

func New() MemDB {
	return MemDB{}
}

var _ KV = (*MemDB)(nil)

func (m MemDB) Set(key []byte, data []byte) {}

func (m MemDB) Get(key []byte) ([]byte, error) {
	return []byte{}, nil
}

func (m MemDB) Del(key []byte) {}
