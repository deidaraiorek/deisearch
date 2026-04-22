package main

import (
	"log"
	"net/http"

	"github.com/deidaraiorek/deisearch/query-engine/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	indexDBPath := "/Users/dangpham/Dev/deisearch/index.db"
	spiderDBPath := "/Users/dangpham/Dev/deisearch/spider.db"
	pagerankDBPath := "/Users/dangpham/Dev/deisearch/pagerank.db"
	embeddingsDBPath := "/Users/dangpham/Dev/deisearch/embeddings.db"

	if err := handler.InitReaders(indexDBPath, spiderDBPath, pagerankDBPath); err != nil {
		log.Fatalf("Failed to initialize readers: %v", err)
	}

	if err := handler.InitSemanticSearch(embeddingsDBPath); err != nil {
		log.Printf("Warning: Semantic search disabled: %v", err)
	}

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/hello", handleHello)
	r.Get("/search", handler.HandleSearch)
	r.Get("/semantic-search", handler.HandleSemanticSearch)

	log.Printf("Starting query engine server on http://localhost:8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleHello(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Hello from Query Engine!"))
}
