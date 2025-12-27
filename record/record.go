package record

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"math"
	"os"
)

var (
	ErrEmptyKey      = errors.New("record with no key is useless")
	ErrPartialWrite  = errors.New("record is in partial write state")
	ErrCorruptRecord = errors.New("record crc is mismatching, corrupted record")
	ErrEncodeInput   = errors.New("encode input invariant failed")
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
	greaterThanUint32MAX := len(key) > math.MaxUint32 || len(val) > math.MaxUint32
	if len(key) == 0 || greaterThanUint32MAX {
		return []byte{}
	}

	keySize := uint32(len(key))
	valSize := uint32(len(val))

	const headerSize = 12 // crc(4) + keySize(4) + valSize(4)
	recordSize := headerSize + keySize + valSize

	buf := make([]byte, recordSize)
	binary.LittleEndian.PutUint32(buf[4:8], keySize)
	binary.LittleEndian.PutUint32(buf[8:12], valSize)

	copy(buf[12:12+keySize], key)

	copy(buf[12+keySize:], val)

	crc := GenerateCRC(keySize, valSize, key, val)
	binary.LittleEndian.PutUint32(buf[0:4], crc)

	return buf
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
	keySize := binary.LittleEndian.Uint32(header[4:8])
	// record without a key is useless
	if keySize == 0 {
		return Record{}, ErrEmptyKey
	}
	valSize := binary.LittleEndian.Uint32(header[8:12])

	recordSize := headerSize + keySize + valSize
	isBiggerThanFileSize := int64(recordSize)+offset > stat.Size()
	if isBiggerThanFileSize {
		// This is a partial write
		// Treat as corruption
		// During index rebuild → stop scanning
		return Record{}, ErrPartialWrite
	}
	offset += int64(headerSize)

	key := make([]byte, keySize)
	n, err := f.ReadAt(key, offset)
	if err != nil {
		return Record{}, err
	}
	bytesRead := n
	offset += int64(keySize)

	val := make([]byte, valSize)
	n, err = f.ReadAt(val, offset)
	if err != nil {
		return Record{}, err
	}
	bytesRead += n
	if bytesRead != int(keySize)+int(valSize) {
		// Partial write
		// Corruption
		return Record{}, ErrPartialWrite
	}
	offset += int64(valSize)

	actualCRC := GenerateCRC(keySize, valSize, key, val)
	if crc != actualCRC {
		return Record{}, ErrCorruptRecord
	}

	return Record{
		Crc:       crc,
		KeySize:   keySize,
		ValueSize: valSize,
		Key:       key,
		Value:     val,
	}, nil
}

func GenerateCRC(keySize, valSize uint32, key, val []byte) uint32 {
	crcTable := crc32.MakeTable(crc32.Castagnoli) // or crc32.IEEE — either is fine
	crcBuf := make([]byte, 8+keySize+valSize)

	binary.LittleEndian.PutUint32(crcBuf[0:4], keySize)
	binary.LittleEndian.PutUint32(crcBuf[4:8], valSize)

	copy(crcBuf[8:8+keySize], key)
	copy(crcBuf[8+keySize:], val)

	return crc32.Checksum(crcBuf, crcTable)
}
