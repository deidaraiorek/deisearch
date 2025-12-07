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
	dbPath := "/Users/dangpham/Dev/deisearch/spider.db"
	logPath := "/Users/dangpham/Dev/deisearch/crawler.log"
	seedURLs := []string{
		"https://www.nature.com/",
		"https://www.britannica.com/",
		"https://www.seriouseats.com/",
		"https://www.zen-habits.net/",
		"https://www.goodreads.com/",
		"https://www.pcgamer.com/",
		"https://www.economist.com/",
		"https://www.lonelyplanet.com/",
		"https://www.metmuseum.org/",
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
		"https://www.bbc.com",
		"https://www.cnn.com",
		"https://www.nytimes.com",
		"https://arxiv.org",
		"https://pubmed.ncbi.nlm.nih.gov",
		"https://www.nih.gov",
		"https://archive.org",
		"https://khanacademy.org",
		"https://www.freecodecamp.org",
		"https://dev.to",
		"https://css-tricks.com",
		"https://uxdesign.cc",
		"https://www.producthunt.com",
		"https://www.stackexchange.com",
		"https://www.researchgate.net",
		"https://www.opensource.org",
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
		Workers:      40,
		RateLimitSec: 0.05,
		MaxPages:     500000,
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
