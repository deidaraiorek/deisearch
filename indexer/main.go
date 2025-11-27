package main

import (
	"io"
	"log"
	"os"

	"github.com/yourusername/deisearch/indexer/internal/indexer"
)

func main() {
	logFile, err := os.OpenFile("indexer.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	spiderDBPath := "/Users/dangpham/Dev/deisearch/search.db"
	indexDBPath := "/Users/dangpham/Dev/deisearch/index.db"
	batchSize := 10000

	log.Printf("Starting indexer...")
	log.Printf("Spider DB: %s", spiderDBPath)
	log.Printf("Index DB: %s", indexDBPath)
	log.Printf("Batch size: %d", batchSize)

	idx, err := indexer.NewIndexer(spiderDBPath, indexDBPath, batchSize)
	if err != nil {
		log.Fatalf("Failed to create indexer: %v", err)
	}
	defer idx.Close()

	if err := idx.IndexAll(); err != nil {
		log.Fatalf("Indexing failed: %v", err)
	}

	log.Printf("Indexing completed successfully!")
}
