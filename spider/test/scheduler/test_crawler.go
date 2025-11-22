package scheduler_test

import (
	"context"
	"log"
	"time"

	"github.com/dangpham/deisearch/spider/internal/scheduler"
	"github.com/dangpham/deisearch/spider/internal/storage"
)

func main() {
	dbPath := "./test_crawler.db"
	seedURLs := []string{
		"https://golang.org",
	}

	log.Println("Initializing database...")
	db, err := storage.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("Creating scheduler...")
	sched := scheduler.New(db, &scheduler.Config{
		Workers:      2,
		RateLimitSec: 1,
		MaxPages:     5,
		UserAgent:    "DeiSearchBot/1.0",
	})

	log.Println("Adding seed URLs...")
	for _, url := range seedURLs {
		if err := sched.AddSeed(url); err != nil {
			log.Printf("Failed to add seed %s: %v", url, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Println("Starting crawler...")
	if err := sched.Start(ctx); err != nil {
		log.Fatalf("Scheduler error: %v", err)
	}

	stats := sched.GetStats()
	log.Printf("Crawling completed! Pages: %d, Queue size: %d", stats["pages_crawled"], stats["queue_size"])
	log.Printf("Database saved to: %s", dbPath)
}
