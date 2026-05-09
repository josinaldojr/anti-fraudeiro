package dataset

const (
	VectorSize                = 14
	LabelLegit                = byte(0)
	LabelFraud                = byte(1)
	BinaryFormatVersionV2     = uint32(2)
	BinaryFormatVersion       = uint32(3)
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
	quantizedMinValue         = float32(-1)
	quantizedMaxValue         = float32(1)
	quantizedOffset           = float32(32767.5)
	quantizedScale            = float32(32767.5)
)

type ReferenceRecord struct {
	Vector []float32 `json:"vector"`
	Label  string    `json:"label"`
}

type VectorStore struct {
	Vectors              []float32
	QuantizedVectors     []uint16
	Labels               []byte
	Count                int
	BucketIndex          [][]uint32
	SecondaryBucketIndex [][]uint32
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
