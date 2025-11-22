package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL: %w", err)
	}

	database := &Database{db: db}

	if err := database.initSchema(); err != nil {
		return nil, err
	}

	return database, nil
}

func (d *Database) initSchema() error {
	schema := `
	-- Pages: Crawled content (URL serves as "seen" marker for deduplication)
	CREATE TABLE IF NOT EXISTS pages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT UNIQUE NOT NULL,
		title TEXT,
		description TEXT,
		content TEXT,
		status_code INTEGER,
		crawled_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_pages_url ON pages(url);

	-- Links: Link graph for PageRank
	-- Stores: from_url links to to_url
	CREATE TABLE IF NOT EXISTS links (
		from_url TEXT,
		to_url TEXT,
		PRIMARY KEY (from_url, to_url)
	);
	`
	_, err := d.db.Exec(schema)
	return err
}

type Page struct {
	URL         string
	Title       string
	Description string
	Content     string
	StatusCode  int
	CrawledAt   time.Time
}

func (d *Database) SavePage(page *Page) error {
	query := `
		INSERT INTO pages (url, title, description, content, status_code, crawled_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			title = excluded.title,
			description = excluded.description,
			content = excluded.content,
			status_code = excluded.status_code,
			crawled_at = excluded.crawled_at
	`

	_, err := d.db.Exec(query,
		page.URL,
		page.Title,
		page.Description,
		page.Content,
		page.StatusCode,
		page.CrawledAt,
	)

	return err
}

func (d *Database) GetPage(url string) (*Page, error) {
	query := "SELECT url, title, description, content, status_code, crawled_at FROM pages WHERE url = ?"

	var page Page
	err := d.db.QueryRow(query, url).Scan(
		&page.URL,
		&page.Title,
		&page.Description,
		&page.Content,
		&page.StatusCode,
		&page.CrawledAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &page, err
}

func (d *Database) LoadAllCrawledURLs() ([]string, error) {
	rows, err := d.db.Query("SELECT url FROM pages")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, rows.Err()
}

func (d *Database) SaveLink(fromURL, toURL string) error {
	_, err := d.db.Exec(`
		INSERT OR IGNORE INTO links (from_url, to_url)
		VALUES (?, ?)
	`, fromURL, toURL)
	return err
}

func (d *Database) SaveLinks(fromURL string, toURLs []string) error {
	if len(toURLs) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR IGNORE INTO links (from_url, to_url) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, toURL := range toURLs {
		if _, err := stmt.Exec(fromURL, toURL); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *Database) GetPageCount() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM pages").Scan(&count)
	return count, err
}

func (d *Database) Close() error {
	return d.db.Close()
}
