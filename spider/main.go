package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dangpham/deisearch/spider/internal/scheduler"
	"github.com/dangpham/deisearch/spider/internal/storage"
)

func main() {
	dbPath := "/Users/dangpham/Dev/deisearch/search.db"
	logPath := "/Users/dangpham/Dev/deisearch/crawler.log"
	seedURLs := []string{
		"https://golang.org",
		"https://go.dev/blog",
		"https://www.youtube.com",
		"https://www.hellointerview.com",
		"https://github.com",
		"https://github.com/donnemartin/system-design-primer",
		"https://news.ycombinator.com",
		"https://www.reddit.com",
		"https://stackoverflow.com",
		"https://www.medium.com",
		"https://www.wikipedia.org",
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	log.Println("Initializing database...")
	db, err := storage.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("Creating scheduler...")
	sched := scheduler.New(db, &scheduler.Config{
		Workers:      20,
		RateLimitSec: 1,
		MaxPages:     750000,
		UserAgent:    "DeiSearchBot/1.0",
	})

	log.Println("Adding seed URLs...")
	for _, url := range seedURLs {
		if err := sched.AddSeed(url); err != nil {
			log.Printf("Failed to add seed %s: %v", url, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("\nShutting down gracefully...")
		cancel()
	}()

	log.Println("Starting crawler...")
	if err := sched.Start(ctx); err != nil {
		log.Fatalf("Scheduler error: %v", err)
	}

	log.Println("Crawling completed!")
	log.Printf("Database saved to: %s", dbPath)
}
