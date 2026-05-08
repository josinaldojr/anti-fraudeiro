package dataset

const (
	VectorSize = 14
	LabelLegit = byte(0)
	LabelFraud = byte(1)
)

type ReferenceRecord struct {
	Vector []float32 `json:"vector"`
	Label  string    `json:"label"`
}

type VectorStore struct {
	Vectors []float32
	Labels  []byte
	Count   int
}
