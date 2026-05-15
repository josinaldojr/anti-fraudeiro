// +build arm64,!noasm

package fraud

func scanQuantizedContiguousSIMD(
	vectors []uint16,
	labels []byte,
	q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15 int32,
	bestDistances *[topK]uint64,
	bestLabels *[topK]byte,
) int {
	// Fallback to Go version for ARM64 to ensure build stability on Mac M1
	// The Go version is already very fast with 16-element alignment.
	return scanQuantizedContiguous(
		vectors, labels,
		q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15,
		bestDistances, bestLabels,
	)
}
