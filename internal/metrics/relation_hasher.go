package metrics

import (
	"crypto/md5"
	"encoding/binary"
)

const relationMod = 4294967295

// labelRelationHash hashes the provided label value and applies the requested modulo.
func labelRelationHash(value string) uint64 {
	sum := md5.Sum([]byte(value))
	segment := sum[8:]
	num := binary.BigEndian.Uint64(segment)
	return num % relationMod
}
