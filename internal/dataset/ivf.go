package dataset

const (
	defaultIVFListCount   = 128
	defaultIVFRefineIters = 4
)

type bucketSummary struct {
	id     uint32
	count  float32
	coords [IVFCoarseDimensions]float32
}

func BuildIVFIndex(store *VectorStore, requestedListCount int) {
	if store == nil || store.Count == 0 || len(store.BucketIndex) == 0 {
		return
	}

	summaries := collectBucketSummaries(store.BucketIndex)
	if len(summaries) == 0 {
		return
	}

	listCount := requestedListCount
	if listCount <= 0 {
		listCount = defaultIVFListCount
	}
	if listCount > len(summaries) {
		listCount = len(summaries)
	}
	if listCount <= 1 {
		store.IVFCentroids = flattenBucketCentroids(summaries[:1])
		store.IVFLists = [][]uint32{{summaries[0].id}}
		return
	}

	centroids := initializeIVFCentroids(summaries, listCount)
	assignments := make([]int, len(summaries))

	for iteration := 0; iteration < defaultIVFRefineIters; iteration++ {
		var sums [][IVFCoarseDimensions]float32
		sums = make([][IVFCoarseDimensions]float32, listCount)
		weights := make([]float32, listCount)

		for index := range summaries {
			assignment := nearestIVFCentroid(summaries[index].coords, centroids)
			assignments[index] = assignment
			weights[assignment] += summaries[index].count
			for dim := 0; dim < IVFCoarseDimensions; dim++ {
				sums[assignment][dim] += summaries[index].coords[dim] * summaries[index].count
			}
		}

		for centroidIndex := 0; centroidIndex < listCount; centroidIndex++ {
			if weights[centroidIndex] == 0 {
				continue
			}
			baseOffset := centroidIndex * IVFCoarseDimensions
			for dim := 0; dim < IVFCoarseDimensions; dim++ {
				centroids[baseOffset+dim] = sums[centroidIndex][dim] / weights[centroidIndex]
			}
		}
	}

	lists := make([][]uint32, listCount)
	for index, summary := range summaries {
		assignment := nearestIVFCentroid(summary.coords, centroids)
		assignments[index] = assignment
		lists[assignment] = append(lists[assignment], summary.id)
	}

	store.IVFCentroids = centroids
	store.IVFLists = lists
}

func collectBucketSummaries(bucketIndex [][]uint32) []bucketSummary {
	summaries := make([]bucketSummary, 0, len(bucketIndex)/8)

	for bucketID, vectorIDs := range bucketIndex {
		if len(vectorIDs) == 0 {
			continue
		}

		amountBucket, hourBucket, dayBucket, txBucket := bucketCoordinatesFromID(bucketID)
		summaries = append(summaries, bucketSummary{
			id:    uint32(bucketID),
			count: float32(len(vectorIDs)),
			coords: [IVFCoarseDimensions]float32{
				bucketCenter(amountBucket, AmountBucketCount),
				bucketCenter(hourBucket, HourBucketCount),
				bucketCenter(dayBucket, DayBucketCount),
				bucketCenter(txBucket, Tx24hBucketCount),
			},
		})
	}

	return summaries
}

func flattenBucketCentroids(summaries []bucketSummary) []float32 {
	centroids := make([]float32, 0, len(summaries)*IVFCoarseDimensions)
	for _, summary := range summaries {
		centroids = append(centroids, summary.coords[:]...)
	}
	return centroids
}

func initializeIVFCentroids(summaries []bucketSummary, listCount int) []float32 {
	centroids := make([]float32, listCount*IVFCoarseDimensions)
	first := densestBucketIndex(summaries)
	copy(centroids[:IVFCoarseDimensions], summaries[first].coords[:])

	selected := make([]bool, len(summaries))
	selected[first] = true

	for centroidIndex := 1; centroidIndex < listCount; centroidIndex++ {
		bestSummaryIndex := -1
		bestScore := float32(-1)
		for summaryIndex := range summaries {
			if selected[summaryIndex] {
				continue
			}

			score := squaredDistance4DToCentroids(summaries[summaryIndex].coords, centroids, centroidIndex) * summaries[summaryIndex].count
			if score > bestScore {
				bestScore = score
				bestSummaryIndex = summaryIndex
			}
		}

		if bestSummaryIndex < 0 {
			bestSummaryIndex = first
		}
		selected[bestSummaryIndex] = true
		copy(centroids[centroidIndex*IVFCoarseDimensions:(centroidIndex+1)*IVFCoarseDimensions], summaries[bestSummaryIndex].coords[:])
	}

	return centroids
}

func densestBucketIndex(summaries []bucketSummary) int {
	bestIndex := 0
	bestCount := summaries[0].count
	for index := 1; index < len(summaries); index++ {
		if summaries[index].count > bestCount {
			bestIndex = index
			bestCount = summaries[index].count
		}
	}
	return bestIndex
}

func nearestIVFCentroid(coords [IVFCoarseDimensions]float32, centroids []float32) int {
	bestIndex := 0
	bestDistance := squaredDistance4D(coords, centroids[0], centroids[1], centroids[2], centroids[3])

	for centroidIndex, baseOffset := 1, IVFCoarseDimensions; baseOffset < len(centroids); centroidIndex, baseOffset = centroidIndex+1, baseOffset+IVFCoarseDimensions {
		distance := squaredDistance4D(
			coords,
			centroids[baseOffset],
			centroids[baseOffset+1],
			centroids[baseOffset+2],
			centroids[baseOffset+3],
		)
		if distance < bestDistance {
			bestDistance = distance
			bestIndex = centroidIndex
		}
	}

	return bestIndex
}

func squaredDistance4DToCentroids(coords [IVFCoarseDimensions]float32, centroids []float32, centroidCount int) float32 {
	bestDistance := squaredDistance4D(coords, centroids[0], centroids[1], centroids[2], centroids[3])

	for centroidIndex, baseOffset := 1, IVFCoarseDimensions; centroidIndex < centroidCount; centroidIndex, baseOffset = centroidIndex+1, baseOffset+IVFCoarseDimensions {
		distance := squaredDistance4D(
			coords,
			centroids[baseOffset],
			centroids[baseOffset+1],
			centroids[baseOffset+2],
			centroids[baseOffset+3],
		)
		if distance < bestDistance {
			bestDistance = distance
		}
	}

	return bestDistance
}

func squaredDistance4D(coords [IVFCoarseDimensions]float32, c0 float32, c1 float32, c2 float32, c3 float32) float32 {
	d0 := coords[0] - c0
	d1 := coords[1] - c1
	d2 := coords[2] - c2
	d3 := coords[3] - c3
	return d0*d0 + d1*d1 + d2*d2 + d3*d3
}

func bucketCoordinatesFromID(bucketID int) (amountBucket int, hourBucket int, dayBucket int, txBucket int) {
	txBucket = bucketID % Tx24hBucketCount
	bucketID /= Tx24hBucketCount
	dayBucket = bucketID % DayBucketCount
	bucketID /= DayBucketCount
	hourBucket = bucketID % HourBucketCount
	bucketID /= HourBucketCount
	amountBucket = bucketID
	return
}

func bucketCenter(bucket int, bucketCount int) float32 {
	return (float32(bucket) + 0.5) / float32(bucketCount)
}
