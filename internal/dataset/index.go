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
		store.BucketPrefixSums = buildBucketPrefixSums(buckets)
		store.SecondaryBucketIndex = nil
		return
	}

	for recordIndex, baseOffset := 0, 0; recordIndex < store.Count; recordIndex, baseOffset = recordIndex+1, baseOffset+VectorSize {
		bucketID := bucketIDFromFloat32(store.Vectors, baseOffset)
		buckets[bucketID] = append(buckets[bucketID], uint32(recordIndex))
	}

	store.BucketIndex = buckets
	store.BucketPrefixSums = buildBucketPrefixSums(buckets)
	store.SecondaryBucketIndex = nil
}

func CountBucketWindowCandidates(
	bucketPrefixSums []uint32,
	amountStart int,
	amountEnd int,
	hourStart int,
	hourEnd int,
	dayStart int,
	dayEnd int,
	txStart int,
	txEnd int,
) int {
	if len(bucketPrefixSums) == 0 {
		return 0
	}

	startAmount := amountStart
	startHour := hourStart
	startDay := dayStart
	startTx := txStart

	endAmount := amountEnd + 1
	endHour := hourEnd + 1
	endDay := dayEnd + 1
	endTx := txEnd + 1

	total := prefixValue(bucketPrefixSums, endAmount, endHour, endDay, endTx) -
		prefixValue(bucketPrefixSums, startAmount, endHour, endDay, endTx) -
		prefixValue(bucketPrefixSums, endAmount, startHour, endDay, endTx) -
		prefixValue(bucketPrefixSums, endAmount, endHour, startDay, endTx) -
		prefixValue(bucketPrefixSums, endAmount, endHour, endDay, startTx) +
		prefixValue(bucketPrefixSums, startAmount, startHour, endDay, endTx) +
		prefixValue(bucketPrefixSums, startAmount, endHour, startDay, endTx) +
		prefixValue(bucketPrefixSums, startAmount, endHour, endDay, startTx) +
		prefixValue(bucketPrefixSums, endAmount, startHour, startDay, endTx) +
		prefixValue(bucketPrefixSums, endAmount, startHour, endDay, startTx) +
		prefixValue(bucketPrefixSums, endAmount, endHour, startDay, startTx) -
		prefixValue(bucketPrefixSums, startAmount, startHour, startDay, endTx) -
		prefixValue(bucketPrefixSums, startAmount, startHour, endDay, startTx) -
		prefixValue(bucketPrefixSums, startAmount, endHour, startDay, startTx) -
		prefixValue(bucketPrefixSums, endAmount, startHour, startDay, startTx) +
		prefixValue(bucketPrefixSums, startAmount, startHour, startDay, startTx)

	return int(total)
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

func ReorderStoreByBucket(store *VectorStore) {
	if store == nil || store.Count == 0 {
		return
	}

	if len(store.BucketIndex) == 0 {
		BuildBucketIndex(store)
	}

	reorderedLabels := make([]byte, store.Count)
	reorderedQuantized := make([]uint16, store.Count*VectorSize)
	bucketMeta := make([]BucketMetadata, BucketIndexCount)

	currentOffset := uint32(0)
	for bucketID, recordIndices := range store.BucketIndex {
		count := uint32(len(recordIndices))
		bucketMeta[bucketID] = BucketMetadata{
			Offset: currentOffset,
			Count:  count,
		}

		for _, recordIndex := range recordIndices {
			reorderedLabels[currentOffset] = store.Labels[recordIndex]

			dstOffset := int(currentOffset) * VectorSize
			if len(store.QuantizedVectors) > 0 {
				srcOffset := int(recordIndex) * VectorSize
				copy(reorderedQuantized[dstOffset:dstOffset+VectorSize], store.QuantizedVectors[srcOffset:srcOffset+VectorSize])
			} else {
				srcOffset := int(recordIndex) * VectorSize
				for i := 0; i < VectorSize; i++ {
					reorderedQuantized[dstOffset+i] = QuantizeComponent(store.Vectors[srcOffset+i])
				}
			}

			currentOffset++
		}
	}

	store.Labels = reorderedLabels
	store.QuantizedVectors = reorderedQuantized
	store.Vectors = nil      // We are moving to quantized only in binary
	store.BucketMeta = bucketMeta
	store.BucketIndex = nil // No longer needed
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

func buildBucketPrefixSums(buckets [][]uint32) []uint32 {
	prefixAmountBucketCount := AmountBucketCount + 1
	prefixHourBucketCount := HourBucketCount + 1
	prefixDayBucketCount := DayBucketCount + 1
	prefixTx24hBucketCount := Tx24hBucketCount + 1

	prefixSums := make([]uint32, prefixAmountBucketCount*prefixHourBucketCount*prefixDayBucketCount*prefixTx24hBucketCount)

	for amountIndex := 0; amountIndex < AmountBucketCount; amountIndex++ {
		for hourIndex := 0; hourIndex < HourBucketCount; hourIndex++ {
			for dayIndex := 0; dayIndex < DayBucketCount; dayIndex++ {
				for txIndex := 0; txIndex < Tx24hBucketCount; txIndex++ {
					prefixSums[prefixIndex(amountIndex+1, hourIndex+1, dayIndex+1, txIndex+1)] =
						uint32(len(buckets[bucketID(amountIndex, hourIndex, dayIndex, txIndex)]))
				}
			}
		}
	}

	for amountIndex := 1; amountIndex <= AmountBucketCount; amountIndex++ {
		for hourIndex := 1; hourIndex <= HourBucketCount; hourIndex++ {
			for dayIndex := 1; dayIndex <= DayBucketCount; dayIndex++ {
				for txIndex := 1; txIndex <= Tx24hBucketCount; txIndex++ {
					index := prefixIndex(amountIndex, hourIndex, dayIndex, txIndex)
					prefixSums[index] += prefixSums[prefixIndex(amountIndex-1, hourIndex, dayIndex, txIndex)]
				}
			}
		}
	}

	for amountIndex := 0; amountIndex <= AmountBucketCount; amountIndex++ {
		for hourIndex := 1; hourIndex <= HourBucketCount; hourIndex++ {
			for dayIndex := 1; dayIndex <= DayBucketCount; dayIndex++ {
				for txIndex := 1; txIndex <= Tx24hBucketCount; txIndex++ {
					index := prefixIndex(amountIndex, hourIndex, dayIndex, txIndex)
					prefixSums[index] += prefixSums[prefixIndex(amountIndex, hourIndex-1, dayIndex, txIndex)]
				}
			}
		}
	}

	for amountIndex := 0; amountIndex <= AmountBucketCount; amountIndex++ {
		for hourIndex := 0; hourIndex <= HourBucketCount; hourIndex++ {
			for dayIndex := 1; dayIndex <= DayBucketCount; dayIndex++ {
				for txIndex := 1; txIndex <= Tx24hBucketCount; txIndex++ {
					index := prefixIndex(amountIndex, hourIndex, dayIndex, txIndex)
					prefixSums[index] += prefixSums[prefixIndex(amountIndex, hourIndex, dayIndex-1, txIndex)]
				}
			}
		}
	}

	for amountIndex := 0; amountIndex <= AmountBucketCount; amountIndex++ {
		for hourIndex := 0; hourIndex <= HourBucketCount; hourIndex++ {
			for dayIndex := 0; dayIndex <= DayBucketCount; dayIndex++ {
				for txIndex := 1; txIndex <= Tx24hBucketCount; txIndex++ {
					index := prefixIndex(amountIndex, hourIndex, dayIndex, txIndex)
					prefixSums[index] += prefixSums[prefixIndex(amountIndex, hourIndex, dayIndex, txIndex-1)]
				}
			}
		}
	}

	return prefixSums
}

func prefixValue(prefixSums []uint32, amountIndex int, hourIndex int, dayIndex int, txIndex int) uint32 {
	return prefixSums[prefixIndex(amountIndex, hourIndex, dayIndex, txIndex)]
}

func prefixIndex(amountIndex int, hourIndex int, dayIndex int, txIndex int) int {
	return (((amountIndex*(HourBucketCount+1))+hourIndex)*(DayBucketCount+1)+dayIndex)*(Tx24hBucketCount+1) + txIndex
}
