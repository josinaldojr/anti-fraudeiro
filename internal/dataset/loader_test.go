package dataset

import (
	"compress/gzip"
	"os"
	"path/filepath"
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
