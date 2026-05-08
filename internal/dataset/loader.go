package dataset

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadVectorStore(path string) (*VectorStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read references file: %w", err)
	}

	var rawRecords []ReferenceRecord
	if err := json.Unmarshal(data, &rawRecords); err != nil {
		return nil, fmt.Errorf("decode references file: %w", err)
	}

	vectors := make([]float32, 0, len(rawRecords)*VectorSize)
	labels := make([]byte, 0, len(rawRecords))

	for index, record := range rawRecords {
		if len(record.Vector) != VectorSize {
			return nil, fmt.Errorf("record %d has %d dimensions, expected %d", index, len(record.Vector), VectorSize)
		}

		vectors = append(vectors, record.Vector...)

		switch record.Label {
		case "legit":
			labels = append(labels, LabelLegit)
		case "fraud":
			labels = append(labels, LabelFraud)
		default:
			return nil, fmt.Errorf("record %d has unknown label %q", index, record.Label)
		}
	}

	return &VectorStore{
		Vectors: vectors,
		Labels:  labels,
		Count:   len(rawRecords),
	}, nil
}
