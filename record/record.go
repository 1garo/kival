package record

import (
	"encoding/binary"
	"os"

	"github.com/1garo/kival/log"
)

// Record is the value encoded or decoded from the db
type Record struct {
	crc       uint32
	keySize   uint32
	valueSize uint32
	key       []byte
	value     []byte
}

// New create a new Record
func New(crc, keySize, valueSize uint32, key, value []byte) Record {
	return Record{
		crc,
		keySize,
		valueSize,
		key,
		value,
	}
}

// Encode encode the record to be inserted into db
func Encode(key, val []byte) []byte {
	return []byte{}
}

// Decode decode the record retrieve from the db
func Decode(
	f *os.File,
	offset int64,
) Record {
	header := make([]byte, log.HeaderSize)
	_, err := f.ReadAt(header, offset)
	if err != nil {
		return Record{}
	}

	crc := binary.LittleEndian.Uint32(header[0:4])
	keyLen := binary.LittleEndian.Uint32(header[4:8])
	// record without a key is useless
	if keyLen == 0 {
		return Record{}
	}
	valLen := binary.LittleEndian.Uint32(header[8:12])

	recordSize := log.HeaderSize + keyLen + valLen
	stat, err := f.Stat()
	if err != nil {
		return Record{}
	}
	if int64(recordSize)+offset >= stat.Size() {
		// This is a partial write
		// Treat as corruption
		// During index rebuild → stop scanning
		return Record{}
	}
	offset += log.HeaderSize

	key := make([]byte, keyLen)
	n, err := f.ReadAt(key, offset)
	if err != nil {
		return Record{}
	}
	bytesRead := n
	offset += int64(keyLen)

	val := make([]byte, valLen)
	n, err = f.ReadAt(val, offset)
	if err != nil {
		return Record{}
	}
	bytesRead += n
	if bytesRead != int(keyLen)+int(valLen) {
		// Partial write
		// Corruption
		return Record{}
	}
	offset += int64(valLen)

	return Record{
		crc:       crc,
		keySize:   keyLen,
		valueSize: valLen,
		key:       key,
		value:     val,
	}
}
