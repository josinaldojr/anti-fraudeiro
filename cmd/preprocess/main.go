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

	count, err := dataset.ConvertJSONToBinary(*inputPath, *outputPath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("wrote %d reference vectors to %s", count, *outputPath)
}
