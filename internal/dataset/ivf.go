package dataset

import "sort"

const (
	defaultIVFListCount   = 128
	defaultIVFRefineIters = 4
	ivfSoftCapacityRatio  = float32(1.10)
	ivfHardCapacityRatio  = float32(1.30)

	ivfDimAmount      = 0
	ivfDimHour        = 1
	ivfDimDay         = 2
	ivfDimTx24h       = 3
	ivfDimAmountVsAvg = 4
	ivfDimRisk        = 5
)

type bucketSummary struct {
	id     uint32
	count  float32
	coords [IVFCoarseDimensions]float32
}

func BuildIVFQueryCoords(amount float32, hour float32, day float32, tx24h float32, amountVsAvg float32, risk float32) [IVFCoarseDimensions]float32 {
	return [IVFCoarseDimensions]float32{
		amount,
		hour,
		day,
		tx24h,
		amountVsAvg,
		risk,
	}
}

func BuildIVFIndex(store *VectorStore, requestedListCount int) {
	if store == nil || store.Count == 0 || len(store.BucketIndex) == 0 {
		return
	}

	summaries := collectBucketSummaries(store)
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
	targetLoad := totalBucketWeight(summaries) / float32(listCount)
	if targetLoad <= 0 {
		targetLoad = 1
	}

	for iteration := 0; iteration < defaultIVFRefineIters; iteration++ {
		loads := assignIVFBucketsConstrained(assignments, summaries, centroids, targetLoad)
		recomputeIVFCentroids(centroids, summaries, assignments, loads)
	}

	lists := make([][]uint32, listCount)
	assignIVFBucketsConstrained(assignments, summaries, centroids, targetLoad)
	for index, summary := range summaries {
		assignment := assignments[index]
		lists[assignment] = append(lists[assignment], summary.id)
	}

	store.IVFCentroids = centroids
	store.IVFLists = lists
}

func collectBucketSummaries(store *VectorStore) []bucketSummary {
	summaries := make([]bucketSummary, 0, len(store.BucketIndex)/8)
	store.IVFBucketSummaries = make([]float32, len(store.BucketIndex)*IVFCoarseDimensions)

	if len(store.QuantizedVectors) > 0 {
		for bucketID, vectorIDs := range store.BucketIndex {
			if len(vectorIDs) == 0 {
				continue
			}

			summary := bucketSummary{
				id:    uint32(bucketID),
				count: float32(len(vectorIDs)),
				coords: summarizeQuantizedBucket(
					store.QuantizedVectors,
					vectorIDs,
				),
			}
			copy(store.IVFBucketSummaries[bucketID*IVFCoarseDimensions:(bucketID+1)*IVFCoarseDimensions], summary.coords[:])
			summaries = append(summaries, summary)
		}
		return summaries
	}

	for bucketID, vectorIDs := range store.BucketIndex {
		if len(vectorIDs) == 0 {
			continue
		}

		summary := bucketSummary{
			id:    uint32(bucketID),
			count: float32(len(vectorIDs)),
			coords: summarizeFloatBucket(
				store.Vectors,
				vectorIDs,
			),
		}
		copy(store.IVFBucketSummaries[bucketID*IVFCoarseDimensions:(bucketID+1)*IVFCoarseDimensions], summary.coords[:])
		summaries = append(summaries, summary)
	}

	return summaries
}

func summarizeFloatBucket(vectors []float32, vectorIDs []uint32) [IVFCoarseDimensions]float32 {
	var sums [IVFCoarseDimensions]float32

	for _, vectorID := range vectorIDs {
		offset := int(vectorID) * VectorSize
		sums[ivfDimAmount] += vectors[offset]
		sums[ivfDimHour] += vectors[offset+3]
		sums[ivfDimDay] += vectors[offset+4]
		sums[ivfDimTx24h] += vectors[offset+8]
		sums[ivfDimAmountVsAvg] += vectors[offset+2]
		sums[ivfDimRisk] += vectors[offset+12]
	}

	scale := 1 / float32(len(vectorIDs))
	for dim := range sums {
		sums[dim] *= scale
	}

	return sums
}

func summarizeQuantizedBucket(vectors []uint16, vectorIDs []uint32) [IVFCoarseDimensions]float32 {
	var sums [IVFCoarseDimensions]float32

	for _, vectorID := range vectorIDs {
		offset := int(vectorID) * VectorSize
		sums[ivfDimAmount] += DequantizeComponent(vectors[offset])
		sums[ivfDimHour] += DequantizeComponent(vectors[offset+3])
		sums[ivfDimDay] += DequantizeComponent(vectors[offset+4])
		sums[ivfDimTx24h] += DequantizeComponent(vectors[offset+8])
		sums[ivfDimAmountVsAvg] += DequantizeComponent(vectors[offset+2])
		sums[ivfDimRisk] += DequantizeComponent(vectors[offset+12])
	}

	scale := 1 / float32(len(vectorIDs))
	for dim := range sums {
		sums[dim] *= scale
	}

	return sums
}

func totalBucketWeight(summaries []bucketSummary) float32 {
	total := float32(0)
	for _, summary := range summaries {
		total += summary.count
	}
	return total
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

			score := squaredDistanceToCentroids(summaries[summaryIndex].coords, centroids, centroidIndex) * summaries[summaryIndex].count
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

func recomputeIVFCentroids(centroids []float32, summaries []bucketSummary, assignments []int, loads []float32) {
	sums := make([][IVFCoarseDimensions]float32, len(loads))
	for index := range summaries {
		assignment := assignments[index]
		for dim := 0; dim < IVFCoarseDimensions; dim++ {
			sums[assignment][dim] += summaries[index].coords[dim] * summaries[index].count
		}
	}

	for centroidIndex := range loads {
		if loads[centroidIndex] == 0 {
			continue
		}
		baseOffset := centroidIndex * IVFCoarseDimensions
		for dim := 0; dim < IVFCoarseDimensions; dim++ {
			centroids[baseOffset+dim] = sums[centroidIndex][dim] / loads[centroidIndex]
		}
	}
}

func assignIVFBucketsConstrained(assignments []int, summaries []bucketSummary, centroids []float32, targetLoad float32) []float32 {
	loads := make([]float32, len(centroids)/IVFCoarseDimensions)
	softCapacity := targetLoad * ivfSoftCapacityRatio
	hardCapacity := targetLoad * ivfHardCapacityRatio

	order := make([]int, len(summaries))
	for index := range summaries {
		order[index] = index
		assignments[index] = -1
	}
	sort.Slice(order, func(left int, right int) bool {
		return summaries[order[left]].count > summaries[order[right]].count
	})

	for _, summaryIndex := range order {
		bestIndex := chooseIVFCentroid(summaries[summaryIndex], centroids, loads, softCapacity, hardCapacity)
		assignments[summaryIndex] = bestIndex
		loads[bestIndex] += summaries[summaryIndex].count
	}

	return loads
}

func chooseIVFCentroid(summary bucketSummary, centroids []float32, loads []float32, softCapacity float32, hardCapacity float32) int {
	nearestIndex := 0
	nearestDistance := squaredDistanceToSlice(summary.coords, centroids, 0)
	nearestProjectedLoad := loads[0] + summary.count
	bestSoftIndex := -1
	bestSoftDistance := float32(0)
	bestHardIndex := -1
	bestHardDistance := float32(0)

	if nearestProjectedLoad <= softCapacity {
		bestSoftIndex = 0
		bestSoftDistance = nearestDistance
	} else if nearestProjectedLoad <= hardCapacity {
		bestHardIndex = 0
		bestHardDistance = nearestDistance
	}

	for centroidIndex, baseOffset := 1, IVFCoarseDimensions; centroidIndex < len(loads); centroidIndex, baseOffset = centroidIndex+1, baseOffset+IVFCoarseDimensions {
		distance := squaredDistanceToSlice(summary.coords, centroids, baseOffset)
		projectedLoad := loads[centroidIndex] + summary.count

		if distance < nearestDistance {
			nearestDistance = distance
			nearestIndex = centroidIndex
			nearestProjectedLoad = projectedLoad
		}

		if projectedLoad <= softCapacity {
			if bestSoftIndex < 0 || distance < bestSoftDistance {
				bestSoftIndex = centroidIndex
				bestSoftDistance = distance
			}
			continue
		}

		if projectedLoad <= hardCapacity {
			if bestHardIndex < 0 || distance < bestHardDistance {
				bestHardIndex = centroidIndex
				bestHardDistance = distance
			}
		}
	}

	if nearestProjectedLoad <= softCapacity {
		return nearestIndex
	}
	if bestSoftIndex >= 0 {
		return bestSoftIndex
	}
	if nearestProjectedLoad <= hardCapacity {
		return nearestIndex
	}
	if bestHardIndex >= 0 {
		return bestHardIndex
	}

	return nearestIndex
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
	bestDistance := squaredDistanceToSlice(coords, centroids, 0)

	for centroidIndex, baseOffset := 1, IVFCoarseDimensions; baseOffset < len(centroids); centroidIndex, baseOffset = centroidIndex+1, baseOffset+IVFCoarseDimensions {
		distance := squaredDistanceToSlice(coords, centroids, baseOffset)
		if distance < bestDistance {
			bestDistance = distance
			bestIndex = centroidIndex
		}
	}

	return bestIndex
}

func squaredDistanceToCentroids(coords [IVFCoarseDimensions]float32, centroids []float32, centroidCount int) float32 {
	bestDistance := squaredDistanceToSlice(coords, centroids, 0)

	for centroidIndex, baseOffset := 1, IVFCoarseDimensions; centroidIndex < centroidCount; centroidIndex, baseOffset = centroidIndex+1, baseOffset+IVFCoarseDimensions {
		distance := squaredDistanceToSlice(coords, centroids, baseOffset)
		if distance < bestDistance {
			bestDistance = distance
		}
	}

	return bestDistance
}

func squaredDistanceToSlice(coords [IVFCoarseDimensions]float32, values []float32, baseOffset int) float32 {
	d0 := coords[0] - values[baseOffset]
	d1 := coords[1] - values[baseOffset+1]
	d2 := coords[2] - values[baseOffset+2]
	d3 := coords[3] - values[baseOffset+3]
	d4 := coords[4] - values[baseOffset+4]
	d5 := coords[5] - values[baseOffset+5]
	return d0*d0 + d1*d1 + d2*d2 + d3*d3 + d4*d4 + d5*d5
}
