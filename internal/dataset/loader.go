package dataset

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func LoadVectorStore(path string) (*VectorStore, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open references file: %w", err)
	}
	defer file.Close()

	reader, closeReader, err := openDatasetReader(file, path)
	if err != nil {
		return nil, err
	}
	defer closeReader()

	decoder := json.NewDecoder(reader)

	startToken, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("read references array start: %w", err)
	}

	delimiter, ok := startToken.(json.Delim)
	if !ok || delimiter != '[' {
		return nil, fmt.Errorf("references file must contain a JSON array")
	}

	vectors := make([]float32, 0, 1024*VectorSize)
	labels := make([]byte, 0, 1024)
	recordIndex := 0

	for decoder.More() {
		var record ReferenceRecord
		if err := decoder.Decode(&record); err != nil {
			return nil, fmt.Errorf("decode record %d: %w", recordIndex, err)
		}

		if len(record.Vector) != VectorSize {
			return nil, fmt.Errorf("record %d has %d dimensions, expected %d", recordIndex, len(record.Vector), VectorSize)
		}

		vectors = append(vectors, record.Vector...)

		switch record.Label {
		case "legit":
			labels = append(labels, LabelLegit)
		case "fraud":
			labels = append(labels, LabelFraud)
		default:
			return nil, fmt.Errorf("record %d has unknown label %q", recordIndex, record.Label)
		}

		recordIndex++
	}

	endToken, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("read references array end: %w", err)
	}

	delimiter, ok = endToken.(json.Delim)
	if !ok || delimiter != ']' {
		return nil, fmt.Errorf("references file has invalid JSON array ending")
	}

	return &VectorStore{
		Vectors: vectors,
		Labels:  labels,
		Count:   recordIndex,
	}, nil
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
