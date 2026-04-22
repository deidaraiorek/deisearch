package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type EmbeddingsDB struct {
	db *sql.DB
}

func NewEmbeddingsDB(dbPath string) (*EmbeddingsDB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open embeddings database: %w", err)
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

	embeddingsDB := &EmbeddingsDB{db: db}

	if err := embeddingsDB.initSchema(); err != nil {
		return nil, err
	}

	return embeddingsDB, nil
}

func (edb *EmbeddingsDB) initSchema() error {
	_, err := edb.db.Exec(Schema)
	return err
}

func (edb *EmbeddingsDB) GetLastIndexedPageID() (int, error) {
	var lastID int
	err := edb.db.QueryRow(
		"SELECT COALESCE(MAX(doc_id), 0) FROM indexed_pages",
	).Scan(&lastID)
	return lastID, err
}

func (edb *EmbeddingsDB) BeginTransaction() (*sql.Tx, error) {
	return edb.db.Begin()
}

func (edb *EmbeddingsDB) SaveEmbeddingWithTx(tx *sql.Tx, docID int, url string, embedding []byte) error {
	if _, err := tx.Exec(
		"INSERT OR IGNORE INTO indexed_pages (doc_id, source_url) VALUES (?, ?)",
		docID, url,
	); err != nil {
		return fmt.Errorf("failed to insert indexed page: %w", err)
	}

	if _, err := tx.Exec(
		"INSERT INTO embeddings (doc_id, embedding) VALUES (?, ?)",
		docID, embedding,
	); err != nil {
		return fmt.Errorf("failed to insert embedding: %w", err)
	}

	return nil
}

func (edb *EmbeddingsDB) UpdateMetadata(key, value string) error {
	_, err := edb.db.Exec(
		"INSERT OR REPLACE INTO embedding_metadata (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)",
		key, value,
	)
	return err
}

func (edb *EmbeddingsDB) Close() error {
	return edb.db.Close()
}
