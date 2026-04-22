package spider

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Page struct {
	ID          int
	URL         string
	Title       string
	Description string
	Content     string
	StatusCode  int
}

type SpiderDB struct {
	db *sql.DB
}

func NewSpiderDB(dbPath string) (*SpiderDB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open spider database: %w", err)
	}

	return &SpiderDB{db: db}, nil
}

func (sdb *SpiderDB) Close() error {
	return sdb.db.Close()
}

func (sdb *SpiderDB) GetPageByID(id int) (*Page, error) {
	page := &Page{}
	err := sdb.db.QueryRow(
		"SELECT id, url, title, description, content, status_code FROM pages WHERE id = ?",
		id,
	).Scan(&page.ID, &page.URL, &page.Title, &page.Description, &page.Content, &page.StatusCode)

	if err != nil {
		return nil, err
	}
	return page, nil
}

func (sdb *SpiderDB) GetPagesAfterID(afterID int, limit int) ([]*Page, error) {
	rows, err := sdb.db.Query(
		"SELECT id, url, title, description, content, status_code FROM pages WHERE id > ? ORDER BY id LIMIT ?",
		afterID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []*Page
	for rows.Next() {
		page := &Page{}
		err := rows.Scan(&page.ID, &page.URL, &page.Title, &page.Description, &page.Content, &page.StatusCode)
		if err != nil {
			return nil, err
		}
		pages = append(pages, page)
	}

	return pages, rows.Err()
}

func (sdb *SpiderDB) GetTotalPageCount() (int, error) {
	var count int
	err := sdb.db.QueryRow("SELECT COUNT(*) FROM pages").Scan(&count)
	return count, err
}
