package record

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"os"
)

var (
	ErrEmptyKey      = errors.New("record with no key is useless")
	ErrPartialWrite  = errors.New("record is in partial write state")
	ErrCorruptRecord = errors.New("record crc is mismatching, corrupted record")
)

// Record is the value encoded or decoded from the db
type Record struct {
	Crc       uint32
	KeySize   uint32
	ValueSize uint32
	Key       []byte
	Value     []byte
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
) (Record, error) {
	stat, err := f.Stat()
	if err != nil {
		return Record{}, nil
	}
	headerSize := uint32(12)

	if offset+int64(headerSize) > stat.Size() {
		return Record{}, ErrPartialWrite
	}

	header := make([]byte, headerSize)
	_, err = f.ReadAt(header, offset)
	if err != nil {
		return Record{}, err
	}

	crc := binary.LittleEndian.Uint32(header[0:4])
	keyLen := binary.LittleEndian.Uint32(header[4:8])
	// record without a key is useless
	if keyLen == 0 {
		return Record{}, ErrEmptyKey
	}
	valLen := binary.LittleEndian.Uint32(header[8:12])

	recordSize := headerSize + keyLen + valLen
	isBiggerThanFileSize := int64(recordSize)+offset > stat.Size()
	if isBiggerThanFileSize {
		// This is a partial write
		// Treat as corruption
		// During index rebuild → stop scanning
		return Record{}, ErrPartialWrite
	}
	offset += int64(headerSize)

	key := make([]byte, keyLen)
	n, err := f.ReadAt(key, offset)
	if err != nil {
		return Record{}, err
	}
	bytesRead := n
	offset += int64(keyLen)

	val := make([]byte, valLen)
	n, err = f.ReadAt(val, offset)
	if err != nil {
		return Record{}, err
	}
	bytesRead += n
	if bytesRead != int(keyLen)+int(valLen) {
		// Partial write
		// Corruption
		return Record{}, ErrPartialWrite
	}
	offset += int64(valLen)

	actualCRC := GenerateCRC(keyLen, valLen, key, val)
	if crc != actualCRC {
		return Record{}, ErrCorruptRecord
	}

	return Record{
		Crc:       crc,
		KeySize:   keyLen,
		ValueSize: valLen,
		Key:       key,
		Value:     val,
	}, nil
}

func GenerateCRC(keyLen, valLen uint32, key, val []byte) uint32 {
	crcTable := crc32.MakeTable(crc32.Castagnoli) // or crc32.IEEE — either is fine
	crcBuf := make([]byte, 8+keyLen+valLen)

	binary.LittleEndian.PutUint32(crcBuf[0:4], keyLen)
	binary.LittleEndian.PutUint32(crcBuf[4:8], valLen)

	copy(crcBuf[8:8+keyLen], key)
	copy(crcBuf[8+keyLen:], val)

	return crc32.Checksum(crcBuf, crcTable)
}
