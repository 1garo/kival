package record

// Format:
//     CRC   (uint32)
//     Timestamp (uint64)
//     KeySize (uint32)
//     ValueSize (uint32)
//     Key   (bytes...)
//     Value (bytes...)

type Record struct {
	Key       []byte
	Value     []byte
	Timestamp int64
}

func (r Record) Encode() {}
func (r Record) Decode() {}
