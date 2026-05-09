package dataset

func BuildBucketIndex(store *VectorStore) {
	if store == nil || store.Count == 0 {
		return
	}

	buckets := make([][]uint32, BucketIndexCount)

	if len(store.QuantizedVectors) > 0 {
		for recordIndex, baseOffset := 0, 0; recordIndex < store.Count; recordIndex, baseOffset = recordIndex+1, baseOffset+VectorSize {
			bucketID := bucketIDFromQuantized(store.QuantizedVectors, baseOffset)
			buckets[bucketID] = append(buckets[bucketID], uint32(recordIndex))
		}
		store.BucketIndex = buckets
		store.SecondaryBucketIndex = nil
		return
	}

	for recordIndex, baseOffset := 0, 0; recordIndex < store.Count; recordIndex, baseOffset = recordIndex+1, baseOffset+VectorSize {
		bucketID := bucketIDFromFloat32(store.Vectors, baseOffset)
		buckets[bucketID] = append(buckets[bucketID], uint32(recordIndex))
	}

	store.BucketIndex = buckets
	store.SecondaryBucketIndex = nil
}

func BuildSecondaryBucketIndex(store *VectorStore) {
	if store == nil || store.Count == 0 {
		return
	}

	buckets := make([][]uint32, SecondaryBucketIndexCount)

	if len(store.QuantizedVectors) > 0 {
		for recordIndex, baseOffset := 0, 0; recordIndex < store.Count; recordIndex, baseOffset = recordIndex+1, baseOffset+VectorSize {
			bucketID := secondaryBucketIDFromQuantized(store.QuantizedVectors, baseOffset)
			buckets[bucketID] = append(buckets[bucketID], uint32(recordIndex))
		}
		store.SecondaryBucketIndex = buckets
		return
	}

	for recordIndex, baseOffset := 0, 0; recordIndex < store.Count; recordIndex, baseOffset = recordIndex+1, baseOffset+VectorSize {
		bucketID := secondaryBucketIDFromFloat32(store.Vectors, baseOffset)
		buckets[bucketID] = append(buckets[bucketID], uint32(recordIndex))
	}

	store.SecondaryBucketIndex = buckets
}

func BucketIDFromQuery(amount float32, hour float32, day float32, tx24h float32) int {
	amountBucket, hourBucket, dayBucket, tx24hBucket := BucketCoordinatesFromQuery(amount, hour, day, tx24h)

	return bucketID(amountBucket, hourBucket, dayBucket, tx24hBucket)
}

func BucketCoordinatesFromQuery(amount float32, hour float32, day float32, tx24h float32) (int, int, int, int) {
	return scalarBucket(amount, AmountBucketCount),
		scalarBucket(hour, HourBucketCount),
		scalarBucket(day, DayBucketCount),
		scalarBucket(tx24h, Tx24hBucketCount)
}

func BucketIDFromCoordinates(amountBucket int, hourBucket int, dayBucket int, tx24hBucket int) int {
	return bucketID(amountBucket, hourBucket, dayBucket, tx24hBucket)
}

func SecondaryBucketCoordinatesFromQuery(amount float32, hour float32, day float32, risk float32) (int, int, int, int) {
	return scalarBucket(amount, AmountBucketCount),
		scalarBucket(hour, HourBucketCount),
		scalarBucket(day, DayBucketCount),
		scalarBucket(risk, RiskBucketCount)
}

func SecondaryBucketIDFromCoordinates(amountBucket int, hourBucket int, dayBucket int, riskBucket int) int {
	return secondaryBucketID(amountBucket, hourBucket, dayBucket, riskBucket)
}

func bucketIDFromFloat32(vectors []float32, baseOffset int) int {
	return bucketID(
		scalarBucket(vectors[baseOffset], AmountBucketCount),
		scalarBucket(vectors[baseOffset+3], HourBucketCount),
		scalarBucket(vectors[baseOffset+4], DayBucketCount),
		scalarBucket(vectors[baseOffset+8], Tx24hBucketCount),
	)
}

func bucketIDFromQuantized(vectors []uint16, baseOffset int) int {
	return bucketID(
		quantizedScalarBucket(vectors[baseOffset], AmountBucketCount),
		quantizedScalarBucket(vectors[baseOffset+3], HourBucketCount),
		quantizedScalarBucket(vectors[baseOffset+4], DayBucketCount),
		quantizedScalarBucket(vectors[baseOffset+8], Tx24hBucketCount),
	)
}

func secondaryBucketIDFromFloat32(vectors []float32, baseOffset int) int {
	return secondaryBucketID(
		scalarBucket(vectors[baseOffset], AmountBucketCount),
		scalarBucket(vectors[baseOffset+3], HourBucketCount),
		scalarBucket(vectors[baseOffset+4], DayBucketCount),
		scalarBucket(vectors[baseOffset+12], RiskBucketCount),
	)
}

func secondaryBucketIDFromQuantized(vectors []uint16, baseOffset int) int {
	return secondaryBucketID(
		quantizedScalarBucket(vectors[baseOffset], AmountBucketCount),
		quantizedScalarBucket(vectors[baseOffset+3], HourBucketCount),
		quantizedScalarBucket(vectors[baseOffset+4], DayBucketCount),
		quantizedScalarBucket(vectors[baseOffset+12], RiskBucketCount),
	)
}

func bucketID(amountBucket int, hourBucket int, dayBucket int, tx24hBucket int) int {
	return (((amountBucket*HourBucketCount)+hourBucket)*DayBucketCount+dayBucket)*Tx24hBucketCount + tx24hBucket
}

func secondaryBucketID(amountBucket int, hourBucket int, dayBucket int, riskBucket int) int {
	return (((amountBucket*HourBucketCount)+hourBucket)*DayBucketCount+dayBucket)*RiskBucketCount + riskBucket
}

func scalarBucket(value float32, bucketCount int) int {
	if value <= 0 {
		return 0
	}
	if value >= 1 {
		return bucketCount - 1
	}

	bucket := int(value * float32(bucketCount))
	if bucket >= bucketCount {
		return bucketCount - 1
	}

	return bucket
}

func quantizedScalarBucket(value uint16, bucketCount int) int {
	return scalarBucket((float32(value)/quantizedScale)-1, bucketCount)
}
