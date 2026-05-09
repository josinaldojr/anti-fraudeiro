package dataset

import (
	"compress/gzip"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadVectorStoreFromGzip(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "references.json.gz")

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create returned error: %v", err)
	}

	gzipWriter := gzip.NewWriter(file)
	_, err = gzipWriter.Write([]byte(`[{"vector":[0,0,0,0,0,-1,-1,0,0,0,0,0,0.5,0],"label":"fraud"},{"vector":[1,1,1,1,1,-1,-1,1,1,1,1,1,0.2,1],"label":"legit"}]`))
	if err != nil {
		t.Fatalf("gzipWriter.Write returned error: %v", err)
	}

	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzipWriter.Close returned error: %v", err)
	}

	if err := file.Close(); err != nil {
		t.Fatalf("file.Close returned error: %v", err)
	}

	store, err := LoadVectorStore(path)
	if err != nil {
		t.Fatalf("LoadVectorStore returned error: %v", err)
	}

	if store.Count != 2 {
		t.Fatalf("store.Count = %d, want 2", store.Count)
	}

	if got := len(store.Vectors); got != 2*VectorSize {
		t.Fatalf("len(store.Vectors) = %d, want %d", got, 2*VectorSize)
	}

	if got := store.Labels[0]; got != LabelFraud {
		t.Fatalf("store.Labels[0] = %d, want %d", got, LabelFraud)
	}

	if got := store.Labels[1]; got != LabelLegit {
		t.Fatalf("store.Labels[1] = %d, want %d", got, LabelLegit)
	}
}

func TestBinaryVectorStoreRoundTrip(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "references.bin")

	expected := &VectorStore{
		Vectors: []float32{
			0, 0, 0, 0, 0, -1, -1, 0, 0, 0, 0, 0, 0.5, 0,
			1, 1, 1, 1, 1, -1, -1, 1, 1, 1, 1, 1, 0.2, 1,
		},
		Labels: []byte{LabelFraud, LabelLegit},
		Count:  2,
	}

	if err := SaveBinaryVectorStore(path, expected); err != nil {
		t.Fatalf("SaveBinaryVectorStore returned error: %v", err)
	}

	actual, err := LoadBinaryVectorStore(path)
	if err != nil {
		t.Fatalf("LoadBinaryVectorStore returned error: %v", err)
	}

	if actual.Count != expected.Count {
		t.Fatalf("actual.Count = %d, want %d", actual.Count, expected.Count)
	}

	if !reflect.DeepEqual(actual.Labels, expected.Labels) {
		t.Fatalf("actual.Labels = %v, want %v", actual.Labels, expected.Labels)
	}

	if got := len(actual.QuantizedVectors); got != expected.Count*VectorSize {
		t.Fatalf("len(actual.QuantizedVectors) = %d, want %d", got, expected.Count*VectorSize)
	}

	for index, value := range expected.Vectors {
		if got := DequantizeComponent(actual.QuantizedVectors[index]); math.Abs(float64(got-value)) > 0.0001 {
			t.Fatalf("dequantized actual[%d] = %f, want %f", index, got, value)
		}
	}
}

func TestConvertJSONToBinary(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "references.json")
	outputPath := filepath.Join(tempDir, "references.bin")

	if err := os.WriteFile(inputPath, []byte(`[{"vector":[0,0,0,0,0,-1,-1,0,0,0,0,0,0.5,0],"label":"fraud"},{"vector":[1,1,1,1,1,-1,-1,1,1,1,1,1,0.2,1],"label":"legit"}]`), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	count, err := ConvertJSONToBinary(inputPath, outputPath)
	if err != nil {
		t.Fatalf("ConvertJSONToBinary returned error: %v", err)
	}

	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}

	store, err := LoadBinaryVectorStore(outputPath)
	if err != nil {
		t.Fatalf("LoadBinaryVectorStore returned error: %v", err)
	}

	if store.Count != 2 {
		t.Fatalf("store.Count = %d, want 2", store.Count)
	}

	if !reflect.DeepEqual(store.Labels, []byte{LabelFraud, LabelLegit}) {
		t.Fatalf("store.Labels = %v, want %v", store.Labels, []byte{LabelFraud, LabelLegit})
	}

	if got := len(store.QuantizedVectors); got != 2*VectorSize {
		t.Fatalf("len(store.QuantizedVectors) = %d, want %d", got, 2*VectorSize)
	}
}
