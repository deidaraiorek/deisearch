package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type IndexReader struct {
	db *sql.DB
}

func NewIndexReader(dbPath string) (*IndexReader, error) {
	connStr := dbPath + "?cache=shared&mode=ro"
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &IndexReader{db: db}, nil
}

func (ir *IndexReader) Close() error {
	return ir.db.Close()
}

func (ir *IndexReader) GetTermIDs(terms []string) ([]int64, error) {
	if len(terms) == 0 {
		return []int64{}, nil
	}

	query := "SELECT term_id FROM terms WHERE term IN ("
	args := make([]interface{}, len(terms))
	for i, term := range terms {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = term
	}
	query += ")"

	rows, err := ir.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query term IDs: %w", err)
	}
	defer rows.Close()

	termIDs := []int64{}
	for rows.Next() {
		var termID int64
		if err := rows.Scan(&termID); err != nil {
			return nil, fmt.Errorf("failed to scan term ID: %w", err)
		}
		termIDs = append(termIDs, termID)
	}

	return termIDs, nil
}

type SearchResult struct {
	DocID int64
	Score float64
}

func (ir *IndexReader) SearchDocuments(termIDs []int64) ([]SearchResult, error) {
	if len(termIDs) == 0 {
		return []SearchResult{}, nil
	}

	if len(termIDs) == 1 {
		query := `
			SELECT doc_id, tfidf as adjusted_score
			FROM postings
			WHERE term_id = ?
			ORDER BY tfidf DESC
			LIMIT 100`

		rows, err := ir.db.Query(query, termIDs[0])
		if err != nil {
			return nil, fmt.Errorf("failed to search documents: %w", err)
		}
		defer rows.Close()

		results := []SearchResult{}
		for rows.Next() {
			var result SearchResult
			if err := rows.Scan(&result.DocID, &result.Score); err != nil {
				return nil, fmt.Errorf("failed to scan result: %w", err)
			}
			results = append(results, result)
		}
		return results, nil
	}

	query := `
		SELECT
			doc_id,
			SUM(tfidf) * (COUNT(DISTINCT term_id) * 1.0 / ?) as adjusted_score
		FROM postings
		WHERE term_id IN (`

	args := make([]interface{}, len(termIDs)+1)
	args[0] = len(termIDs)

	for i, termID := range termIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i+1] = termID
	}

	query += `)
		GROUP BY doc_id
		ORDER BY adjusted_score DESC
		LIMIT 100`

	rows, err := ir.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}
	defer rows.Close()

	results := []SearchResult{}
	for rows.Next() {
		var result SearchResult
		if err := rows.Scan(&result.DocID, &result.Score); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}
