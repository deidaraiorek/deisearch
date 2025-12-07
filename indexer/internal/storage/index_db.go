package storage

import (
	"database/sql"
	"fmt"
	"math"

	_ "github.com/mattn/go-sqlite3"
)

type IndexDB struct {
	db *sql.DB
}

func NewIndexDB(dbPath string) (*IndexDB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL: %w", err)
	}

	if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		return nil, fmt.Errorf("failed to set synchronous mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA cache_size=10000"); err != nil {
		return nil, fmt.Errorf("failed to set cache size: %w", err)
	}

	if _, err := db.Exec("PRAGMA temp_store=MEMORY"); err != nil {
		return nil, fmt.Errorf("failed to set temp store: %w", err)
	}

	if _, err := db.Exec("PRAGMA mmap_size=30000000"); err != nil {
		return nil, fmt.Errorf("failed to set mmap size: %w", err)
	}

	indexDB := &IndexDB{
		db: db,
	}

	if err := indexDB.initSchema(); err != nil {
		return nil, err
	}

	return indexDB, nil
}

func (idb *IndexDB) initSchema() error {
	_, err := idb.db.Exec(Schema)
	return err
}

func (idb *IndexDB) IsPageIndexed(pageID int) (bool, error) {
	var exists bool
	err := idb.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM indexed_pages WHERE doc_id = ?)",
		pageID,
	).Scan(&exists)
	return exists, err
}

func (idb *IndexDB) GetLastIndexedPageID() (int, error) {
	var lastID int
	err := idb.db.QueryRow(
		"SELECT COALESCE(MAX(doc_id), 0) FROM indexed_pages",
	).Scan(&lastID)
	return lastID, err
}

func (idb *IndexDB) MarkPageAsIndexed(pageID int, url string) error {
	_, err := idb.db.Exec(
		"INSERT OR IGNORE INTO indexed_pages (doc_id, source_url) VALUES (?, ?)",
		pageID, url,
	)
	return err
}

func (idb *IndexDB) GetIndexedPageCount() (int, error) {
	var count int
	err := idb.db.QueryRow("SELECT COUNT(*) FROM indexed_pages").Scan(&count)
	return count, err
}

func (idb *IndexDB) GetOrCreateTermID(term string) (int64, error) {
	var termID int64
	err := idb.db.QueryRow(
		"SELECT term_id FROM terms WHERE term = ?",
		term,
	).Scan(&termID)

	if err == sql.ErrNoRows {
		result, err := idb.db.Exec(
			"INSERT INTO terms (term, document_frequency) VALUES (?, 0)",
			term,
		)
		if err != nil {
			return 0, err
		}
		return result.LastInsertId()
	}
	return termID, err
}

func (idb *IndexDB) SavePosting(termID int64, docID int, termFreq int, tf, tfidf float64) error {
	_, err := idb.db.Exec(
		"INSERT OR REPLACE INTO postings (term_id, doc_id, term_frequency, tf, tfidf) VALUES (?, ?, ?, ?, ?)",
		termID, docID, termFreq, tf, tfidf,
	)
	return err
}

func (idb *IndexDB) SaveDocStats(docID int, docLength int, uniqueTerms int) error {
	_, err := idb.db.Exec(
		"INSERT OR REPLACE INTO doc_stats (doc_id, doc_length, unique_terms) VALUES (?, ?, ?)",
		docID, docLength, uniqueTerms,
	)
	return err
}

func (idb *IndexDB) UpdateDocumentFrequency(termID int64) error {
	_, err := idb.db.Exec(
		"UPDATE terms SET document_frequency = document_frequency + 1 WHERE term_id = ?",
		termID,
	)
	return err
}

func (idb *IndexDB) BeginTransaction() (*sql.Tx, error) {
	return idb.db.Begin()
}

func (idb *IndexDB) SetMetadata(key, value string) error {
	_, err := idb.db.Exec(
		"INSERT OR REPLACE INTO index_metadata (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)",
		key, value,
	)
	return err
}

func (idb *IndexDB) GetMetadata(key string) (string, error) {
	var value string
	err := idb.db.QueryRow(
		"SELECT value FROM index_metadata WHERE key = ?",
		key,
	).Scan(&value)
	return value, err
}

func (idb *IndexDB) Close() error {
	return idb.db.Close()
}

func (idb *IndexDB) RecalculateTFIDF() error {
	totalDocs, err := idb.GetIndexedPageCount()
	if err != nil {
		return fmt.Errorf("failed to get total document count: %w", err)
	}

	tx, err := idb.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Step 1: Calculate IDF values in Go and update terms table
	// IDF = log(total_docs / document_frequency)
	rows, err := tx.Query("SELECT term_id, document_frequency FROM terms WHERE document_frequency > 0")
	if err != nil {
		return fmt.Errorf("failed to query terms: %w", err)
	}

	updateStmt, err := tx.Prepare("UPDATE terms SET idf = ? WHERE term_id = ?")
	if err != nil {
		rows.Close()
		return fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer updateStmt.Close()

	for rows.Next() {
		var termID int64
		var docFreq int
		if err := rows.Scan(&termID, &docFreq); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan term: %w", err)
		}

		// Calculate IDF = log(total_docs / document_frequency)
		idf := math.Log(float64(totalDocs) / float64(docFreq))

		if _, err := updateStmt.Exec(idf, termID); err != nil {
			rows.Close()
			return fmt.Errorf("failed to update IDF for term %d: %w", termID, err)
		}
	}
	rows.Close()

	// Step 2: Update TF and TF-IDF in postings using a single JOIN-based query
	// TF = term_frequency / doc_length
	// TF-IDF = TF * IDF
	_, err = tx.Exec(`
		UPDATE postings
		SET
			tf = CAST(term_frequency AS REAL) / CAST(doc_stats.doc_length AS REAL),
			tfidf = (CAST(term_frequency AS REAL) / CAST(doc_stats.doc_length AS REAL)) * terms.idf
		FROM doc_stats, terms
		WHERE postings.doc_id = doc_stats.doc_id
		  AND postings.term_id = terms.term_id
	`)
	if err != nil {
		return fmt.Errorf("failed to update TF-IDF values: %w", err)
	}

	if _, err := tx.Exec("UPDATE index_metadata SET value = ? WHERE key = 'total_documents'", totalDocs); err != nil {
		return fmt.Errorf("failed to update total documents: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

type PreparedStatements struct {
	insertPage     *sql.Stmt
	insertDocStats *sql.Stmt
	getTerm        *sql.Stmt
	insertTerm     *sql.Stmt
	updateDF       *sql.Stmt
	insertPosting  *sql.Stmt
}

func (idb *IndexDB) PrepareStatements(tx *sql.Tx) (*PreparedStatements, error) {
	stmts := &PreparedStatements{}
	var err error

	stmts.insertPage, err = tx.Prepare("INSERT OR IGNORE INTO indexed_pages (doc_id, source_url) VALUES (?, ?)")
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insertPage: %w", err)
	}

	stmts.insertDocStats, err = tx.Prepare("INSERT OR REPLACE INTO doc_stats (doc_id, doc_length, unique_terms) VALUES (?, ?, ?)")
	if err != nil {
		stmts.insertPage.Close()
		return nil, fmt.Errorf("failed to prepare insertDocStats: %w", err)
	}

	stmts.getTerm, err = tx.Prepare("SELECT term_id FROM terms WHERE term = ?")
	if err != nil {
		stmts.insertPage.Close()
		stmts.insertDocStats.Close()
		return nil, fmt.Errorf("failed to prepare getTerm: %w", err)
	}

	stmts.insertTerm, err = tx.Prepare("INSERT INTO terms (term, document_frequency) VALUES (?, 1)")
	if err != nil {
		stmts.insertPage.Close()
		stmts.insertDocStats.Close()
		stmts.getTerm.Close()
		return nil, fmt.Errorf("failed to prepare insertTerm: %w", err)
	}

	stmts.updateDF, err = tx.Prepare("UPDATE terms SET document_frequency = document_frequency + 1 WHERE term_id = ?")
	if err != nil {
		stmts.insertPage.Close()
		stmts.insertDocStats.Close()
		stmts.getTerm.Close()
		stmts.insertTerm.Close()
		return nil, fmt.Errorf("failed to prepare updateDF: %w", err)
	}

	stmts.insertPosting, err = tx.Prepare("INSERT INTO postings (term_id, doc_id, term_frequency) VALUES (?, ?, ?)")
	if err != nil {
		stmts.insertPage.Close()
		stmts.insertDocStats.Close()
		stmts.getTerm.Close()
		stmts.insertTerm.Close()
		stmts.updateDF.Close()
		return nil, fmt.Errorf("failed to prepare insertPosting: %w", err)
	}

	return stmts, nil
}

func (ps *PreparedStatements) Close() {
	if ps.insertPage != nil {
		ps.insertPage.Close()
	}
	if ps.insertDocStats != nil {
		ps.insertDocStats.Close()
	}
	if ps.getTerm != nil {
		ps.getTerm.Close()
	}
	if ps.insertTerm != nil {
		ps.insertTerm.Close()
	}
	if ps.updateDF != nil {
		ps.updateDF.Close()
	}
	if ps.insertPosting != nil {
		ps.insertPosting.Close()
	}
}

func (idb *IndexDB) SaveDocumentWithStatements(stmts *PreparedStatements, docID int, url string, termFreqs map[string]int, docLength int) error {
	_, err := stmts.insertPage.Exec(docID, url)
	if err != nil {
		return fmt.Errorf("failed to mark page as indexed: %w", err)
	}

	_, err = stmts.insertDocStats.Exec(docID, docLength, len(termFreqs))
	if err != nil {
		return fmt.Errorf("failed to save doc stats: %w", err)
	}

	for term, freq := range termFreqs {
		var termID int64

		err := stmts.getTerm.QueryRow(term).Scan(&termID)
		if err == sql.ErrNoRows {
			result, err := stmts.insertTerm.Exec(term)
			if err != nil {
				return fmt.Errorf("failed to insert term %q: %w", term, err)
			}
			termID, err = result.LastInsertId()
			if err != nil {
				return err
			}
		} else if err != nil {
			return fmt.Errorf("failed to query term %q: %w", term, err)
		} else {
			_, err = stmts.updateDF.Exec(termID)
			if err != nil {
				return fmt.Errorf("failed to update document frequency for term %q: %w", term, err)
			}
		}

		_, err = stmts.insertPosting.Exec(termID, docID, freq)
		if err != nil {
			return fmt.Errorf("failed to insert posting for term %q: %w", term, err)
		}
	}

	return nil
}

func (idb *IndexDB) SaveDocumentInTransaction(tx *sql.Tx, docID int, url string, termFreqs map[string]int, docLength int) error {
	_, err := tx.Exec(
		"INSERT OR IGNORE INTO indexed_pages (doc_id, source_url) VALUES (?, ?)",
		docID, url,
	)
	if err != nil {
		return fmt.Errorf("failed to mark page as indexed: %w", err)
	}

	_, err = tx.Exec(
		"INSERT OR REPLACE INTO doc_stats (doc_id, doc_length, unique_terms) VALUES (?, ?, ?)",
		docID, docLength, len(termFreqs),
	)
	if err != nil {
		return fmt.Errorf("failed to save doc stats: %w", err)
	}

	getTermStmt, err := tx.Prepare("SELECT term_id FROM terms WHERE term = ?")
	if err != nil {
		return err
	}
	defer getTermStmt.Close()

	insertTermStmt, err := tx.Prepare("INSERT INTO terms (term, document_frequency) VALUES (?, 1)")
	if err != nil {
		return err
	}
	defer insertTermStmt.Close()

	updateDFStmt, err := tx.Prepare("UPDATE terms SET document_frequency = document_frequency + 1 WHERE term_id = ?")
	if err != nil {
		return err
	}
	defer updateDFStmt.Close()

	insertPostingStmt, err := tx.Prepare("INSERT INTO postings (term_id, doc_id, term_frequency) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer insertPostingStmt.Close()

	for term, freq := range termFreqs {
		var termID int64

		err := getTermStmt.QueryRow(term).Scan(&termID)
		if err == sql.ErrNoRows {
			result, err := insertTermStmt.Exec(term)
			if err != nil {
				return fmt.Errorf("failed to insert term %q: %w", term, err)
			}
			termID, err = result.LastInsertId()
			if err != nil {
				return err
			}
		} else if err != nil {
			return fmt.Errorf("failed to query term %q: %w", term, err)
		} else {
			_, err = updateDFStmt.Exec(termID)
			if err != nil {
				return fmt.Errorf("failed to update document frequency for term %q: %w", term, err)
			}
		}

		_, err = insertPostingStmt.Exec(termID, docID, freq)
		if err != nil {
			return fmt.Errorf("failed to insert posting for term %q: %w", term, err)
		}
	}

	return nil
}
