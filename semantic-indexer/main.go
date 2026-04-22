package main

import (
	"io"
	"log"
	"os"

	"github.com/deidaraiorek/deisearch/semantic-indexer/internal/indexer"
)

func main() {
	logFile, err := os.OpenFile("semantic_indexer.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	spiderDBPath := "/Users/dangpham/Dev/deisearch/spider.db"
	embeddingsDBPath := "/Users/dangpham/Dev/deisearch/embeddings.db"
	batchSize := 32

	log.Printf("Starting semantic indexer...")
	log.Printf("Spider DB: %s", spiderDBPath)
	log.Printf("Embeddings DB: %s", embeddingsDBPath)
	log.Printf("Batch size: %d", batchSize)

	idx, err := indexer.NewIndexer(spiderDBPath, embeddingsDBPath, batchSize)
	if err != nil {
		log.Fatalf("Failed to create indexer: %v", err)
	}
	defer idx.Close()

	if err := idx.IndexAll(); err != nil {
		log.Fatalf("Indexing failed: %v", err)
	}

	log.Printf("Semantic indexing completed successfully!")
}
