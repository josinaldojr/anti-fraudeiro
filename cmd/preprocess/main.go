package main

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

func main() {
	inputPath := flag.String("input", filepath.Join("resources", dataset.CompressedReferenceFile), "path to the source references dataset (.json or .json.gz)")
	outputPath := flag.String("output", filepath.Join("resources", dataset.BinaryReferenceFile), "path to the compact binary output")
	flag.Parse()

	var count int
	switch filepath.Ext(*inputPath) {
	case ".bin":
		store, err := dataset.LoadBinaryVectorStore(*inputPath)
		if err != nil {
			log.Fatal(err)
		}
		defer store.Close()

		dataset.ReorderStoreByBucket(store)

		if err := dataset.SaveBinaryVectorStore(*outputPath, store); err != nil {
			log.Fatal(err)
		}
		count = store.Count
	default:
		store, err := dataset.LoadVectorStore(*inputPath)
		if err != nil {
			log.Fatal(err)
		}
		defer store.Close()

		dataset.ReorderStoreByBucket(store)

		if err := dataset.SaveBinaryVectorStore(*outputPath, store); err != nil {
			log.Fatal(err)
		}
		count = store.Count
	}

	log.Printf("wrote %d reference vectors to %s", count, *outputPath)
}
