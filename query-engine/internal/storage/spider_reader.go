package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type SpiderReader struct {
	db *sql.DB
}

func NewSpiderReader(dbPath string) (*SpiderReader, error) {
	connStr := dbPath + "?cache=shared&mode=ro"
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open spider database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping spider database: %w", err)
	}

	return &SpiderReader{db: db}, nil
}

func (sr *SpiderReader) Close() error {
	return sr.db.Close()
}

type PageInfo struct {
	ID          int64
	URL         string
	Title       string
	Description string
	Content     string
}

func (sr *SpiderReader) GetPagesByIDs(docIDs []int64) (map[int64]PageInfo, error) {
	if len(docIDs) == 0 {
		return map[int64]PageInfo{}, nil
	}

	query := "SELECT id, url, title, description, content FROM pages WHERE id IN ("
	args := make([]interface{}, len(docIDs))
	for i, docID := range docIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = docID
	}
	query += ")"

	rows, err := sr.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pages: %w", err)
	}
	defer rows.Close()

	pages := make(map[int64]PageInfo)
	for rows.Next() {
		var page PageInfo
		if err := rows.Scan(&page.ID, &page.URL, &page.Title, &page.Description, &page.Content); err != nil {
			return nil, fmt.Errorf("failed to scan page: %w", err)
		}
		pages[page.ID] = page
	}

	return pages, nil
}
