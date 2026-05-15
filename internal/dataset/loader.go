package dataset

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unsafe"
)

func LoadVectorStore(path string) (*VectorStore, error) {
	if filepath.Ext(path) == ".bin" {
		return LoadBinaryVectorStore(path)
	}

	decoder, closeDecoder, err := openDatasetDecoder(path)
	if err != nil {
		return nil, err
	}
	defer closeDecoder()

	if err := expectJSONArrayStart(decoder); err != nil {
		return nil, err
	}

	vectors := make([]float32, 0, 1024*VectorSize)
	labels := make([]byte, 0, 1024)
	recordIndex := 0

	for decoder.More() {
		var record ReferenceRecord
		if err := decoder.Decode(&record); err != nil {
			return nil, fmt.Errorf("decode record %d: %w", recordIndex, err)
		}

		if len(record.Vector) != 14 {
			return nil, fmt.Errorf("record %d has %d dimensions, expected 14", recordIndex, len(record.Vector))
		}

		vectors = append(vectors, record.Vector...)
		vectors = append(vectors, 0, 0) // Pad 14 to 16 dimensions

		label, err := parseLabel(record.Label)
		if err != nil {
			return nil, fmt.Errorf("record %d %w", recordIndex, err)
		}
		labels = append(labels, label)

		recordIndex++
	}

	if err := expectJSONArrayEnd(decoder); err != nil {
		return nil, err
	}

	store := &VectorStore{
		Vectors: vectors,
		Labels:  labels,
		Count:   recordIndex,
	}
	BuildBucketIndex(store)
	return store, nil
}

func LoadBinaryVectorStore(path string) (*VectorStore, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open binary references file: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	header, err := readBinaryHeader(reader)
	if err != nil {
		return nil, err
	}

	count := int(header.Count)
	labels := make([]byte, count)

	switch header.Version {
	case BinaryFormatVersionV2:
		vectors := make([]float32, count*VectorSize)
		for recordIndex := 0; recordIndex < count; recordIndex++ {
			label, err := reader.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("read binary label %d: %w", recordIndex, err)
			}
			labels[recordIndex] = label

			offset := recordIndex * VectorSize
			if err := binary.Read(reader, binary.LittleEndian, vectors[offset:offset+VectorSize]); err != nil {
				return nil, fmt.Errorf("read binary vector %d: %w", recordIndex, err)
			}
		}

		store := &VectorStore{
			Vectors: vectors,
			Labels:  labels,
			Count:   count,
		}
		BuildBucketIndex(store)
		return store, nil
	case BinaryFormatVersion:
		return loadMappedQuantizedVectorStore(path, header)
	case BinaryFormatVersionV3:
		quantizedVectors := make([]uint16, count*VectorSize)
		for recordIndex := 0; recordIndex < count; recordIndex++ {
			label, err := reader.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("read binary label %d: %w", recordIndex, err)
			}
			labels[recordIndex] = label

			offset := recordIndex * VectorSize
			if err := binary.Read(reader, binary.LittleEndian, quantizedVectors[offset:offset+VectorSize]); err != nil {
				return nil, fmt.Errorf("read quantized vector %d: %w", recordIndex, err)
			}
		}

		store := &VectorStore{
			QuantizedVectors: quantizedVectors,
			Labels:           labels,
			Count:            count,
		}
		BuildBucketIndex(store)
		return store, nil
	default:
		return nil, fmt.Errorf("unsupported binary references version %d", header.Version)
	}
}

func SaveBinaryVectorStore(path string, store *VectorStore) error {
	if store == nil {
		return fmt.Errorf("vector store is nil")
	}
	if len(store.Labels) != store.Count {
		return fmt.Errorf("label count %d does not match store count %d", len(store.Labels), store.Count)
	}
	if len(store.QuantizedVectors) == 0 && len(store.Vectors) != store.Count*VectorSize {
		return fmt.Errorf("vector length %d does not match expected %d", len(store.Vectors), store.Count*VectorSize)
	}
	if len(store.QuantizedVectors) > 0 && len(store.QuantizedVectors) != store.Count*VectorSize {
		return fmt.Errorf("quantized vector length %d does not match expected %d", len(store.QuantizedVectors), store.Count*VectorSize)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create binary references file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	header := binaryHeader{
		Magic:      binaryMagic,
		Version:    BinaryFormatVersion,
		VectorSize: VectorSize,
		Count:      uint64(store.Count),
	}

	if err := binary.Write(writer, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("write binary header: %w", err)
	}

	if header.Version >= 5 {
		meta := store.BucketMeta
		if len(meta) == 0 {
			meta = make([]BucketMetadata, BucketIndexCount)
		}
		if err := binary.Write(writer, binary.LittleEndian, meta); err != nil {
			return fmt.Errorf("write bucket meta: %w", err)
		}
	}

	if _, err := writer.Write(store.Labels); err != nil {
		return fmt.Errorf("write binary labels: %w", err)
	}

	if (len(store.Labels)+len(store.BucketMeta)*binary.Size(BucketMetadata{}))%2 != 0 {
		if err := writer.WriteByte(0); err != nil {
			return fmt.Errorf("write binary padding: %w", err)
		}
	}

	if len(store.QuantizedVectors) > 0 {
		if err := binary.Write(writer, binary.LittleEndian, store.QuantizedVectors); err != nil {
			return fmt.Errorf("write quantized vectors: %w", err)
		}
	} else {
		for recordIndex := 0; recordIndex < store.Count; recordIndex++ {
			offset := recordIndex * VectorSize
			var quantizedBuffer [VectorSize]uint16
			for index, value := range store.Vectors[offset : offset+VectorSize] {
				quantizedBuffer[index] = QuantizeComponent(value)
			}
			if err := binary.Write(writer, binary.LittleEndian, quantizedBuffer); err != nil {
				return fmt.Errorf("write quantized vector %d: %w", recordIndex, err)
			}
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush binary references file: %w", err)
	}

	return nil
}

func ConvertJSONToBinary(inputPath string, outputPath string) (int, error) {
	store, err := LoadVectorStore(inputPath)
	if err != nil {
		return 0, err
	}
	defer store.Close()

	ReorderStoreByBucket(store)

	if err := SaveBinaryVectorStore(outputPath, store); err != nil {
		return 0, err
	}

	return store.Count, nil
}

func openDatasetReader(file *os.File, path string) (io.Reader, func() error, error) {
	if filepath.Ext(path) != ".gz" {
		return file, func() error { return nil }, nil
	}

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, nil, fmt.Errorf("open gzip references file: %w", err)
	}

	return gzipReader, gzipReader.Close, nil
}

var binaryMagic = [8]byte{'A', 'F', 'R', 'D', 'B', 'I', 'N', '1'}

type binaryHeader struct {
	Magic      [8]byte
	Version    uint32
	VectorSize uint32
	Count      uint64
}

func readBinaryHeader(reader io.Reader) (binaryHeader, error) {
	var header binaryHeader

	if err := binary.Read(reader, binary.LittleEndian, &header); err != nil {
		return binaryHeader{}, fmt.Errorf("read binary header: %w", err)
	}

	if header.Magic != binaryMagic {
		return binaryHeader{}, fmt.Errorf("invalid binary references magic")
	}

	if header.VectorSize != VectorSize {
		return binaryHeader{}, fmt.Errorf("binary references vector size %d, expected %d", header.VectorSize, VectorSize)
	}

	return header, nil
}

func openDatasetDecoder(path string) (*json.Decoder, func() error, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open references file: %w", err)
	}

	reader, closeReader, err := openDatasetReader(file, path)
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}

	closeAll := func() error {
		readerErr := closeReader()
		fileErr := file.Close()
		if readerErr != nil {
			return readerErr
		}
		return fileErr
	}

	return json.NewDecoder(reader), closeAll, nil
}

func expectJSONArrayStart(decoder *json.Decoder) error {
	startToken, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("read references array start: %w", err)
	}

	delimiter, ok := startToken.(json.Delim)
	if !ok || delimiter != '[' {
		return fmt.Errorf("references file must contain a JSON array")
	}

	return nil
}

func expectJSONArrayEnd(decoder *json.Decoder) error {
	endToken, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("read references array end: %w", err)
	}

	delimiter, ok := endToken.(json.Delim)
	if !ok || delimiter != ']' {
		return fmt.Errorf("references file has invalid JSON array ending")
	}

	return nil
}

func parseLabel(value string) (byte, error) {
	switch value {
	case "legit":
		return LabelLegit, nil
	case "fraud":
		return LabelFraud, nil
	default:
		return 0, fmt.Errorf("has unknown label %q", value)
	}
}

func binaryRecordFromReference(record ReferenceRecord) (byte, [VectorSize]float32, error) {
	if len(record.Vector) != 14 {
		return 0, [VectorSize]float32{}, fmt.Errorf("has %d dimensions, expected 14", len(record.Vector))
	}

	label, err := parseLabel(record.Label)
	if err != nil {
		return 0, [VectorSize]float32{}, err
	}

	var vector [VectorSize]float32
	copy(vector[:14], record.Vector)
	vector[14] = 0 // Padding
	vector[15] = 0 // Padding
	return label, vector, nil
}

func writePlaceholderHeader(file *os.File) error {
	header := binaryHeader{
		Magic:      binaryMagic,
		Version:    BinaryFormatVersionV3,
		VectorSize: VectorSize,
		Count:      0,
	}

	if err := binary.Write(file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("write binary placeholder header: %w", err)
	}

	return nil
}

func writeFinalHeader(file *os.File, count int) error {
	header := binaryHeader{
		Magic:      binaryMagic,
		Version:    BinaryFormatVersionV3,
		VectorSize: VectorSize,
		Count:      uint64(count),
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek binary header: %w", err)
	}

	if err := binary.Write(file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("write binary final header: %w", err)
	}

	return nil
}

func loadMappedQuantizedVectorStore(path string, header binaryHeader) (*VectorStore, error) {
	mappedData, err := mmapFile(path)
	if err != nil {
		return nil, err
	}

	count := int(header.Count)
	labelsOffset := binary.Size(binaryHeader{})
	var bucketMeta []BucketMetadata

	if header.Version >= 5 {
		metaCount := BucketIndexCount
		metaSize := metaCount * binary.Size(BucketMetadata{})
		metaBytes := mappedData[labelsOffset : labelsOffset+metaSize]
		bucketMeta = unsafe.Slice((*BucketMetadata)(unsafe.Pointer(&metaBytes[0])), metaCount)
		labelsOffset += metaSize
	}

	vectorsOffset := labelsOffset + count
	if vectorsOffset%2 != 0 {
		vectorsOffset++
	}

	vectorBytesLength := count * VectorSize * 2
	expectedSize := vectorsOffset + vectorBytesLength
	if len(mappedData) < expectedSize {
		_ = unmapFile(mappedData)
		return nil, fmt.Errorf("binary references file is truncated: got %d bytes, need %d", len(mappedData), expectedSize)
	}

	labels := mappedData[labelsOffset : labelsOffset+count]
	vectorBytes := mappedData[vectorsOffset:expectedSize]
	quantizedVectors := bytesAsUint16(vectorBytes)

	store := &VectorStore{
		QuantizedVectors: quantizedVectors,
		Labels:           labels,
		Count:            count,
		BucketMeta:       bucketMeta,
		mappedData:       mappedData,
	}
	BuildBucketIndex(store)
	return store, nil
}

func buildBucketPrefixSumsFromMeta(meta []BucketMetadata) []uint32 {
	prefixAmountBucketCount := AmountBucketCount + 1
	prefixHourBucketCount := HourBucketCount + 1
	prefixDayBucketCount := DayBucketCount + 1
	prefixTx24hBucketCount := Tx24hBucketCount + 1

	prefixSums := make([]uint32, prefixAmountBucketCount*prefixHourBucketCount*prefixDayBucketCount*prefixTx24hBucketCount)

	for amountIndex := 0; amountIndex < AmountBucketCount; amountIndex++ {
		for hourIndex := 0; hourIndex < HourBucketCount; hourIndex++ {
			for dayIndex := 0; dayIndex < DayBucketCount; dayIndex++ {
				for txIndex := 0; txIndex < Tx24hBucketCount; txIndex++ {
					bucketID := (((amountIndex*HourBucketCount)+hourIndex)*DayBucketCount+dayIndex)*Tx24hBucketCount + txIndex
					prefixSums[prefixIndex(amountIndex+1, hourIndex+1, dayIndex+1, txIndex+1)] = meta[bucketID].Count
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

func bytesAsUint16(data []byte) []uint16 {
	if len(data) == 0 {
		return nil
	}

	return unsafe.Slice((*uint16)(unsafe.Pointer(&data[0])), len(data)/2)
}
