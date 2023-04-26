package utils

import "encoding/binary"

func U64ToBytes(i uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, i)
	return buf
}

func BytesToU64(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}
