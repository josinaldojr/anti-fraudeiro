// +build amd64,!noasm

package fraud

import (
	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

func scanQuantizedContiguousSIMD(
	vectors []uint16,
	labels []byte,
	q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15 int32,
	bestDistances *[topK]uint64,
	bestLabels *[topK]byte,
) int {
	count := len(labels)
	if count == 0 {
		return 0
	}

	qArr := [16]int32{q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15}
	threshold := bestDistances[topK-1]
	
	const batchSize = 128
	var dists [batchSize]uint64
	
	for i := 0; i < count; i += batchSize {
		end := i + batchSize
		if end > count {
			end = count
		}
		
		n := end - i
		batchVectors := vectors[i*dataset.VectorSize : end*dataset.VectorSize]
		
		DistancesAVX2(&qArr, batchVectors, dists[:n])
		
		for j := 0; j < n; j++ {
			d := dists[j]
			if d < threshold {
				insertTopKUint64(d, labels[i+j], bestDistances, bestLabels)
				threshold = bestDistances[topK-1]
			}
		}
	}

	return count
}
