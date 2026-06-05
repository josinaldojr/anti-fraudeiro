package dataset

func BuildBucketIndex(store *VectorStore) {
	if store == nil || store.Count == 0 {
		return
	}
	if len(store.BucketMeta) == BucketIndexCount {
		store.BucketIndex = nil
		store.BucketPrefixSums = buildBucketPrefixSumsFromMeta(store.BucketMeta)
		return
	}

	// If BucketMeta exists but is wrong size (e.g., v5 binary loaded with v6 code),
	// clear it so the search falls back to BucketIndex-based scanning.
	if len(store.BucketMeta) > 0 {
		store.BucketMeta = nil
	}

	buckets := make([][]uint32, BucketIndexCount)

	if len(store.QuantizedVectors) > 0 {
		for recordIndex, baseOffset := 0, 0; recordIndex < store.Count; recordIndex, baseOffset = recordIndex+1, baseOffset+VectorSize {
			bucketID := bucketIDFromQuantized(store.QuantizedVectors, baseOffset)
			buckets[bucketID] = append(buckets[bucketID], uint32(recordIndex))
		}
		store.BucketIndex = buckets
		store.BucketPrefixSums = buildBucketPrefixSums(buckets)
		return
	}

	for recordIndex, baseOffset := 0, 0; recordIndex < store.Count; recordIndex, baseOffset = recordIndex+1, baseOffset+VectorSize {
		bucketID := bucketIDFromFloat32(store.Vectors, baseOffset)
		buckets[bucketID] = append(buckets[bucketID], uint32(recordIndex))
	}

	store.BucketIndex = buckets
	store.BucketPrefixSums = buildBucketPrefixSums(buckets)
}

func CountBucketWindowCandidates(
	bucketPrefixSums []uint32,
	amountStart int,
	amountEnd int,
	minutesStart int,
	minutesEnd int,
	kmHomeStart int,
	kmHomeEnd int,
	txStart int,
	txEnd int,
	amountVsAvgStart int,
	amountVsAvgEnd int,
	mccRiskStart int,
	mccRiskEnd int,
) int {
	if len(bucketPrefixSums) == 0 {
		return 0
	}

	sa, sm, sk, st, sv, sr := amountStart, minutesStart, kmHomeStart, txStart, amountVsAvgStart, mccRiskStart
	ea, em, ek, et, ev, er := amountEnd+1, minutesEnd+1, kmHomeEnd+1, txEnd+1, amountVsAvgEnd+1, mccRiskEnd+1

	total := prefixValue(bucketPrefixSums, ea, em, ek, et, ev, er) -
		prefixValue(bucketPrefixSums, sa, em, ek, et, ev, er) -
		prefixValue(bucketPrefixSums, ea, sm, ek, et, ev, er) -
		prefixValue(bucketPrefixSums, ea, em, sk, et, ev, er) -
		prefixValue(bucketPrefixSums, ea, em, ek, st, ev, er) -
		prefixValue(bucketPrefixSums, ea, em, ek, et, sv, er) -
		prefixValue(bucketPrefixSums, ea, em, ek, et, ev, sr) +
		prefixValue(bucketPrefixSums, sa, sm, ek, et, ev, er) +
		prefixValue(bucketPrefixSums, sa, em, sk, et, ev, er) +
		prefixValue(bucketPrefixSums, sa, em, ek, st, ev, er) +
		prefixValue(bucketPrefixSums, sa, em, ek, et, sv, er) +
		prefixValue(bucketPrefixSums, sa, em, ek, et, ev, sr) +
		prefixValue(bucketPrefixSums, ea, sm, sk, et, ev, er) +
		prefixValue(bucketPrefixSums, ea, sm, ek, st, ev, er) +
		prefixValue(bucketPrefixSums, ea, sm, ek, et, sv, er) +
		prefixValue(bucketPrefixSums, ea, sm, ek, et, ev, sr) +
		prefixValue(bucketPrefixSums, ea, em, sk, st, ev, er) +
		prefixValue(bucketPrefixSums, ea, em, sk, et, sv, er) +
		prefixValue(bucketPrefixSums, ea, em, sk, et, ev, sr) +
		prefixValue(bucketPrefixSums, ea, em, ek, st, sv, er) +
		prefixValue(bucketPrefixSums, ea, em, ek, st, ev, sr) +
		prefixValue(bucketPrefixSums, ea, em, ek, et, sv, sr) -
		prefixValue(bucketPrefixSums, sa, sm, sk, et, ev, er) -
		prefixValue(bucketPrefixSums, sa, sm, ek, st, ev, er) -
		prefixValue(bucketPrefixSums, sa, sm, ek, et, sv, er) -
		prefixValue(bucketPrefixSums, sa, sm, ek, et, ev, sr) -
		prefixValue(bucketPrefixSums, sa, em, sk, st, ev, er) -
		prefixValue(bucketPrefixSums, sa, em, sk, et, sv, er) -
		prefixValue(bucketPrefixSums, sa, em, sk, et, ev, sr) -
		prefixValue(bucketPrefixSums, sa, em, ek, st, sv, er) -
		prefixValue(bucketPrefixSums, sa, em, ek, st, ev, sr) -
		prefixValue(bucketPrefixSums, sa, em, ek, et, sv, sr) -
		prefixValue(bucketPrefixSums, ea, sm, sk, st, ev, er) -
		prefixValue(bucketPrefixSums, ea, sm, sk, et, sv, er) -
		prefixValue(bucketPrefixSums, ea, sm, sk, et, ev, sr) -
		prefixValue(bucketPrefixSums, ea, sm, ek, st, sv, er) -
		prefixValue(bucketPrefixSums, ea, sm, ek, st, ev, sr) -
		prefixValue(bucketPrefixSums, ea, sm, ek, et, sv, sr) -
		prefixValue(bucketPrefixSums, ea, em, sk, st, sv, er) -
		prefixValue(bucketPrefixSums, ea, em, sk, st, ev, sr) -
		prefixValue(bucketPrefixSums, ea, em, sk, et, sv, sr) -
		prefixValue(bucketPrefixSums, ea, em, ek, st, sv, sr) +
		prefixValue(bucketPrefixSums, sa, sm, sk, st, ev, er) +
		prefixValue(bucketPrefixSums, sa, sm, sk, et, sv, er) +
		prefixValue(bucketPrefixSums, sa, sm, sk, et, ev, sr) +
		prefixValue(bucketPrefixSums, sa, sm, ek, st, sv, er) +
		prefixValue(bucketPrefixSums, sa, sm, ek, st, ev, sr) +
		prefixValue(bucketPrefixSums, sa, sm, ek, et, sv, sr) +
		prefixValue(bucketPrefixSums, sa, em, sk, st, sv, er) +
		prefixValue(bucketPrefixSums, sa, em, sk, st, ev, sr) +
		prefixValue(bucketPrefixSums, sa, em, sk, et, sv, sr) +
		prefixValue(bucketPrefixSums, sa, em, ek, st, sv, sr) +
		prefixValue(bucketPrefixSums, ea, sm, sk, st, sv, er) +
		prefixValue(bucketPrefixSums, ea, sm, sk, st, ev, sr) +
		prefixValue(bucketPrefixSums, ea, sm, sk, et, sv, sr) +
		prefixValue(bucketPrefixSums, ea, sm, ek, st, sv, sr) +
		prefixValue(bucketPrefixSums, ea, em, sk, st, sv, sr) -
		prefixValue(bucketPrefixSums, sa, sm, sk, st, sv, er) -
		prefixValue(bucketPrefixSums, sa, sm, sk, st, ev, sr) -
		prefixValue(bucketPrefixSums, sa, sm, sk, et, sv, sr) -
		prefixValue(bucketPrefixSums, sa, sm, ek, st, sv, sr) -
		prefixValue(bucketPrefixSums, sa, em, sk, st, sv, sr) -
		prefixValue(bucketPrefixSums, ea, sm, sk, st, sv, sr) +
		prefixValue(bucketPrefixSums, sa, sm, sk, st, sv, sr)

	return int(total)
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
		count := uint16(len(recordIndices))
		var fraudCount uint16
		for _, recordIndex := range recordIndices {
			reorderedLabels[currentOffset] = store.Labels[recordIndex]
			if store.Labels[recordIndex] == LabelFraud {
				fraudCount++
			}

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
		bucketMeta[bucketID] = BucketMetadata{
			Offset:     currentOffset - uint32(count),
			Count:      count,
			FraudCount: fraudCount,
		}
	}

	store.Labels = reorderedLabels
	store.QuantizedVectors = reorderedQuantized
	store.Vectors = nil
	store.BucketMeta = bucketMeta
	store.BucketIndex = nil
}

func BucketIDFromQuery(amount, minutes, kmHome, tx24h, amountVsAvg, mccRisk float32) int {
	amountBucket, minutesBucket, kmHomeBucket, tx24hBucket, amountVsAvgBucket, mccRiskBucket :=
		BucketCoordinatesFromQuery(amount, minutes, kmHome, tx24h, amountVsAvg, mccRisk)
	return bucketID(amountBucket, minutesBucket, kmHomeBucket, tx24hBucket, amountVsAvgBucket, mccRiskBucket)
}

func BucketCoordinatesFromQuery(amount, minutes, kmHome, tx24h, amountVsAvg, mccRisk float32) (int, int, int, int, int, int) {
	return scalarBucket(amount, AmountBucketCount),
		scalarBucket(minutes, MinutesSinceLastCount),
		scalarBucket(kmHome, KMFromHomeCount),
		scalarBucket(tx24h, Tx24hBucketCount),
		scalarBucket(amountVsAvg, AmountVsAvgBucketCount),
		scalarBucket(mccRisk, MCCRiskBucketCount)
}

func BucketIDFromCoordinates(amountBucket, minutesBucket, kmHomeBucket, tx24hBucket, amountVsAvgBucket, mccRiskBucket int) int {
	return bucketID(amountBucket, minutesBucket, kmHomeBucket, tx24hBucket, amountVsAvgBucket, mccRiskBucket)
}

func bucketIDFromFloat32(vectors []float32, baseOffset int) int {
	return bucketID(
		scalarBucket(vectors[baseOffset], AmountBucketCount),
		scalarBucket(vectors[baseOffset+5], MinutesSinceLastCount),
		scalarBucket(vectors[baseOffset+7], KMFromHomeCount),
		scalarBucket(vectors[baseOffset+8], Tx24hBucketCount),
		scalarBucket(vectors[baseOffset+2], AmountVsAvgBucketCount),
		scalarBucket(vectors[baseOffset+12], MCCRiskBucketCount),
	)
}

func bucketIDFromQuantized(vectors []uint16, baseOffset int) int {
	return bucketID(
		quantizedScalarBucket(vectors[baseOffset], AmountBucketCount),
		quantizedScalarBucket(vectors[baseOffset+5], MinutesSinceLastCount),
		quantizedScalarBucket(vectors[baseOffset+7], KMFromHomeCount),
		quantizedScalarBucket(vectors[baseOffset+8], Tx24hBucketCount),
		quantizedScalarBucket(vectors[baseOffset+2], AmountVsAvgBucketCount),
		quantizedScalarBucket(vectors[baseOffset+12], MCCRiskBucketCount),
	)
}

func bucketID(amountBucket, minutesBucket, kmHomeBucket, tx24hBucket, amountVsAvgBucket, mccRiskBucket int) int {
	return ((((((amountBucket*MinutesSinceLastCount)+minutesBucket)*KMFromHomeCount+kmHomeBucket)*
		Tx24hBucketCount+tx24hBucket)*AmountVsAvgBucketCount+amountVsAvgBucket)*MCCRiskBucketCount + mccRiskBucket)
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
	pa := AmountBucketCount + 1
	pm := MinutesSinceLastCount + 1
	pk := KMFromHomeCount + 1
	pt := Tx24hBucketCount + 1
	pv := AmountVsAvgBucketCount + 1
	pr := MCCRiskBucketCount + 1

	prefixSums := make([]uint32, pa*pm*pk*pt*pv*pr)

	for a := 0; a < AmountBucketCount; a++ {
		for m := 0; m < MinutesSinceLastCount; m++ {
			for k := 0; k < KMFromHomeCount; k++ {
				for t := 0; t < Tx24hBucketCount; t++ {
					for v := 0; v < AmountVsAvgBucketCount; v++ {
						for r := 0; r < MCCRiskBucketCount; r++ {
							prefixSums[prefixIndex(a+1, m+1, k+1, t+1, v+1, r+1)] =
								uint32(len(buckets[bucketID(a, m, k, t, v, r)]))
						}
					}
				}
			}
		}
	}

	for a := 1; a <= AmountBucketCount; a++ {
		for m := 1; m <= MinutesSinceLastCount; m++ {
			for k := 1; k <= KMFromHomeCount; k++ {
				for t := 1; t <= Tx24hBucketCount; t++ {
					for v := 1; v <= AmountVsAvgBucketCount; v++ {
						for r := 1; r <= MCCRiskBucketCount; r++ {
							idx := prefixIndex(a, m, k, t, v, r)
							prefixSums[idx] += prefixSums[prefixIndex(a-1, m, k, t, v, r)]
						}
					}
				}
			}
		}
	}

	for a := 0; a <= AmountBucketCount; a++ {
		for m := 1; m <= MinutesSinceLastCount; m++ {
			for k := 1; k <= KMFromHomeCount; k++ {
				for t := 1; t <= Tx24hBucketCount; t++ {
					for v := 1; v <= AmountVsAvgBucketCount; v++ {
						for r := 1; r <= MCCRiskBucketCount; r++ {
							idx := prefixIndex(a, m, k, t, v, r)
							prefixSums[idx] += prefixSums[prefixIndex(a, m-1, k, t, v, r)]
						}
					}
				}
			}
		}
	}

	for a := 0; a <= AmountBucketCount; a++ {
		for m := 0; m <= MinutesSinceLastCount; m++ {
			for k := 1; k <= KMFromHomeCount; k++ {
				for t := 1; t <= Tx24hBucketCount; t++ {
					for v := 1; v <= AmountVsAvgBucketCount; v++ {
						for r := 1; r <= MCCRiskBucketCount; r++ {
							idx := prefixIndex(a, m, k, t, v, r)
							prefixSums[idx] += prefixSums[prefixIndex(a, m, k-1, t, v, r)]
						}
					}
				}
			}
		}
	}

	for a := 0; a <= AmountBucketCount; a++ {
		for m := 0; m <= MinutesSinceLastCount; m++ {
			for k := 0; k <= KMFromHomeCount; k++ {
				for t := 1; t <= Tx24hBucketCount; t++ {
					for v := 1; v <= AmountVsAvgBucketCount; v++ {
						for r := 1; r <= MCCRiskBucketCount; r++ {
							idx := prefixIndex(a, m, k, t, v, r)
							prefixSums[idx] += prefixSums[prefixIndex(a, m, k, t-1, v, r)]
						}
					}
				}
			}
		}
	}

	for a := 0; a <= AmountBucketCount; a++ {
		for m := 0; m <= MinutesSinceLastCount; m++ {
			for k := 0; k <= KMFromHomeCount; k++ {
				for t := 0; t <= Tx24hBucketCount; t++ {
					for v := 1; v <= AmountVsAvgBucketCount; v++ {
						for r := 1; r <= MCCRiskBucketCount; r++ {
							idx := prefixIndex(a, m, k, t, v, r)
							prefixSums[idx] += prefixSums[prefixIndex(a, m, k, t, v-1, r)]
						}
					}
				}
			}
		}
	}

	for a := 0; a <= AmountBucketCount; a++ {
		for m := 0; m <= MinutesSinceLastCount; m++ {
			for k := 0; k <= KMFromHomeCount; k++ {
				for t := 0; t <= Tx24hBucketCount; t++ {
					for v := 0; v <= AmountVsAvgBucketCount; v++ {
						for r := 1; r <= MCCRiskBucketCount; r++ {
							idx := prefixIndex(a, m, k, t, v, r)
							prefixSums[idx] += prefixSums[prefixIndex(a, m, k, t, v, r-1)]
						}
					}
				}
			}
		}
	}

	return prefixSums
}

func prefixValue(prefixSums []uint32, amountIndex, minutesIndex, kmHomeIndex, txIndex, amountVsAvgIndex, mccRiskIndex int) uint32 {
	return prefixSums[prefixIndex(amountIndex, minutesIndex, kmHomeIndex, txIndex, amountVsAvgIndex, mccRiskIndex)]
}

func prefixIndex(amountIndex, minutesIndex, kmHomeIndex, txIndex, amountVsAvgIndex, mccRiskIndex int) int {
	return ((((((amountIndex*(MinutesSinceLastCount+1))+minutesIndex)*(KMFromHomeCount+1)+kmHomeIndex)*
		(Tx24hBucketCount+1)+txIndex)*(AmountVsAvgBucketCount+1)+amountVsAvgIndex)*(MCCRiskBucketCount+1) + mccRiskIndex)
}

func buildBucketPrefixSumsFromMeta(meta []BucketMetadata) []uint32 {
	pa := AmountBucketCount + 1
	pm := MinutesSinceLastCount + 1
	pk := KMFromHomeCount + 1
	pt := Tx24hBucketCount + 1
	pv := AmountVsAvgBucketCount + 1
	pr := MCCRiskBucketCount + 1

	prefixSums := make([]uint32, pa*pm*pk*pt*pv*pr)

	for a := 0; a < AmountBucketCount; a++ {
		for m := 0; m < MinutesSinceLastCount; m++ {
			for k := 0; k < KMFromHomeCount; k++ {
				for t := 0; t < Tx24hBucketCount; t++ {
					for v := 0; v < AmountVsAvgBucketCount; v++ {
						for r := 0; r < MCCRiskBucketCount; r++ {
							cellID := bucketID(a, m, k, t, v, r)
							prefixSums[prefixIndex(a+1, m+1, k+1, t+1, v+1, r+1)] = uint32(meta[cellID].Count)
						}
					}
				}
			}
		}
	}

	for a := 1; a <= AmountBucketCount; a++ {
		for m := 1; m <= MinutesSinceLastCount; m++ {
			for k := 1; k <= KMFromHomeCount; k++ {
				for t := 1; t <= Tx24hBucketCount; t++ {
					for v := 1; v <= AmountVsAvgBucketCount; v++ {
						for r := 1; r <= MCCRiskBucketCount; r++ {
							idx := prefixIndex(a, m, k, t, v, r)
							prefixSums[idx] += prefixSums[prefixIndex(a-1, m, k, t, v, r)]
						}
					}
				}
			}
		}
	}

	for a := 0; a <= AmountBucketCount; a++ {
		for m := 1; m <= MinutesSinceLastCount; m++ {
			for k := 1; k <= KMFromHomeCount; k++ {
				for t := 1; t <= Tx24hBucketCount; t++ {
					for v := 1; v <= AmountVsAvgBucketCount; v++ {
						for r := 1; r <= MCCRiskBucketCount; r++ {
							idx := prefixIndex(a, m, k, t, v, r)
							prefixSums[idx] += prefixSums[prefixIndex(a, m-1, k, t, v, r)]
						}
					}
				}
			}
		}
	}

	for a := 0; a <= AmountBucketCount; a++ {
		for m := 0; m <= MinutesSinceLastCount; m++ {
			for k := 1; k <= KMFromHomeCount; k++ {
				for t := 1; t <= Tx24hBucketCount; t++ {
					for v := 1; v <= AmountVsAvgBucketCount; v++ {
						for r := 1; r <= MCCRiskBucketCount; r++ {
							idx := prefixIndex(a, m, k, t, v, r)
							prefixSums[idx] += prefixSums[prefixIndex(a, m, k-1, t, v, r)]
						}
					}
				}
			}
		}
	}

	for a := 0; a <= AmountBucketCount; a++ {
		for m := 0; m <= MinutesSinceLastCount; m++ {
			for k := 0; k <= KMFromHomeCount; k++ {
				for t := 1; t <= Tx24hBucketCount; t++ {
					for v := 1; v <= AmountVsAvgBucketCount; v++ {
						for r := 1; r <= MCCRiskBucketCount; r++ {
							idx := prefixIndex(a, m, k, t, v, r)
							prefixSums[idx] += prefixSums[prefixIndex(a, m, k, t-1, v, r)]
						}
					}
				}
			}
		}
	}

	for a := 0; a <= AmountBucketCount; a++ {
		for m := 0; m <= MinutesSinceLastCount; m++ {
			for k := 0; k <= KMFromHomeCount; k++ {
				for t := 0; t <= Tx24hBucketCount; t++ {
					for v := 1; v <= AmountVsAvgBucketCount; v++ {
						for r := 1; r <= MCCRiskBucketCount; r++ {
							idx := prefixIndex(a, m, k, t, v, r)
							prefixSums[idx] += prefixSums[prefixIndex(a, m, k, t, v-1, r)]
						}
					}
				}
			}
		}
	}

	for a := 0; a <= AmountBucketCount; a++ {
		for m := 0; m <= MinutesSinceLastCount; m++ {
			for k := 0; k <= KMFromHomeCount; k++ {
				for t := 0; t <= Tx24hBucketCount; t++ {
					for v := 0; v <= AmountVsAvgBucketCount; v++ {
						for r := 1; r <= MCCRiskBucketCount; r++ {
							idx := prefixIndex(a, m, k, t, v, r)
							prefixSums[idx] += prefixSums[prefixIndex(a, m, k, t, v, r-1)]
						}
					}
				}
			}
		}
	}

	return prefixSums
}
