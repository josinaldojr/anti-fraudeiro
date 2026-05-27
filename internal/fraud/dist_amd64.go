// +build amd64,!noasm

package fraud

// DistancesAVX2 calculates squared Euclidean distances for multiple quantized vectors using AVX2.
// query: 16 int32 values.
// vectors: N * 16 uint16 values.
// distances: N uint64 values.
//go:noescape
func DistancesAVX2(query *[16]int32, vectors []uint16, distances []uint64)

//go:noescape
func hasAVX2() bool

