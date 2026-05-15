package dataset

const (
	VectorSize                = 16
	LabelLegit                = byte(0)
	LabelFraud                = byte(1)
	BinaryFormatVersionV2     = uint32(2)
	BinaryFormatVersionV3     = uint32(3)
	BinaryFormatVersionV4     = uint32(4)
	BinaryFormatVersion       = uint32(5)
	BinaryReferenceFile       = "references.bin"
	CompressedReferenceFile   = "references.json.gz"
	ExampleReferenceFile      = "example-references.json"
	AmountBucketCount         = 32
	HourBucketCount           = 24
	DayBucketCount            = 7
	Tx24hBucketCount          = 16
	RiskBucketCount           = 8
	BucketIndexCount          = AmountBucketCount * HourBucketCount * DayBucketCount * Tx24hBucketCount
	SecondaryBucketIndexCount = AmountBucketCount * HourBucketCount * DayBucketCount * RiskBucketCount
	IVFCoarseDimensions       = 6
	quantizedMinValue         = float32(-1)
	quantizedMaxValue         = float32(1)
	quantizedOffset           = float32(32767.5)
	quantizedScale            = float32(32767.5)
)

type ReferenceRecord struct {
	Vector []float32 `json:"vector"`
	Label  string    `json:"label"`
}

type BucketMetadata struct {
	Offset uint32
	Count  uint32
}

type VectorStore struct {
	Vectors              []float32
	QuantizedVectors     []uint16
	Labels               []byte
	Count                int
	BucketMeta           []BucketMetadata
	BucketIndex          [][]uint32
	BucketPrefixSums     []uint32
	SecondaryBucketIndex [][]uint32
	IVFCentroids         []float32
	IVFBucketSummaries   []float32
	IVFLists             [][]uint32
	mappedData           []byte
}

func (store *VectorStore) Close() error {
	if store == nil || len(store.mappedData) == 0 {
		return nil
	}

	mappedData := store.mappedData
	store.mappedData = nil
	store.Vectors = nil
	store.QuantizedVectors = nil
	store.Labels = nil
	store.BucketMeta = nil
	store.BucketIndex = nil
	store.BucketPrefixSums = nil
	store.SecondaryBucketIndex = nil
	store.IVFCentroids = nil
	store.IVFBucketSummaries = nil
	store.IVFLists = nil

	return unmapFile(mappedData)
}

func QuantizeComponent(value float32) uint16 {
	if value <= quantizedMinValue {
		return 0
	}
	if value >= quantizedMaxValue {
		return ^uint16(0)
	}

	return uint16(((value + 1) * quantizedScale) + 0.5)
}

func DequantizeComponent(value uint16) float32 {
	return (float32(value) / quantizedScale) - 1
}
