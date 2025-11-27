package storage_test

import (
	"os"
	"testing"

	"github.com/yourusername/deisearch/indexer/internal/storage"
)

func TestNewIndexDB(t *testing.T) {
	dbPath := "/tmp/test_index.db"
	defer os.Remove(dbPath)

	db, err := storage.NewIndexDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create index DB: %v", err)
	}
	defer db.Close()

	count, err := db.GetIndexedPageCount()
	if err != nil {
		t.Fatalf("Failed to get indexed page count: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 indexed pages, got %d", count)
	}
}

func TestIsPageIndexed(t *testing.T) {
	dbPath := "/tmp/test_index_2.db"
	defer os.Remove(dbPath)

	db, err := storage.NewIndexDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create index DB: %v", err)
	}
	defer db.Close()

	indexed, err := db.IsPageIndexed(1)
	if err != nil {
		t.Fatalf("Failed to check if page is indexed: %v", err)
	}
	if indexed {
		t.Error("Expected page 1 to not be indexed")
	}

	err = db.MarkPageAsIndexed(1, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to mark page as indexed: %v", err)
	}

	indexed, err = db.IsPageIndexed(1)
	if err != nil {
		t.Fatalf("Failed to check if page is indexed: %v", err)
	}
	if !indexed {
		t.Error("Expected page 1 to be indexed")
	}
}

func TestGetOrCreateTermID(t *testing.T) {
	dbPath := "/tmp/test_index_3.db"
	defer os.Remove(dbPath)

	db, err := storage.NewIndexDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create index DB: %v", err)
	}
	defer db.Close()

	termID1, err := db.GetOrCreateTermID("machine")
	if err != nil {
		t.Fatalf("Failed to create term: %v", err)
	}

	termID2, err := db.GetOrCreateTermID("machine")
	if err != nil {
		t.Fatalf("Failed to get term: %v", err)
	}

	if termID1 != termID2 {
		t.Errorf("Expected same term ID, got %d and %d", termID1, termID2)
	}

	termID3, err := db.GetOrCreateTermID("learning")
	if err != nil {
		t.Fatalf("Failed to create term: %v", err)
	}

	if termID1 == termID3 {
		t.Error("Expected different term IDs for different terms")
	}
}

func TestSaveDocumentInTransaction(t *testing.T) {
	dbPath := "/tmp/test_index_4.db"
	defer os.Remove(dbPath)

	db, err := storage.NewIndexDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create index DB: %v", err)
	}
	defer db.Close()

	termFreqs := map[string]int{
		"machine":  3,
		"learning": 2,
		"ai":       1,
	}

	tx, err := db.BeginTransaction()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	err = db.SaveDocumentInTransaction(tx, 1, "https://example.com/ml", termFreqs, 6)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to save document: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	indexed, err := db.IsPageIndexed(1)
	if err != nil {
		t.Fatalf("Failed to check if page is indexed: %v", err)
	}
	if !indexed {
		t.Error("Expected page to be indexed")
	}

	count, err := db.GetIndexedPageCount()
	if err != nil {
		t.Fatalf("Failed to get indexed page count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 indexed page, got %d", count)
	}
}

func TestGetLastIndexedPageID(t *testing.T) {
	dbPath := "/tmp/test_index_5.db"
	defer os.Remove(dbPath)

	db, err := storage.NewIndexDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create index DB: %v", err)
	}
	defer db.Close()

	lastID, err := db.GetLastIndexedPageID()
	if err != nil {
		t.Fatalf("Failed to get last indexed page ID: %v", err)
	}
	if lastID != 0 {
		t.Errorf("Expected last ID to be 0, got %d", lastID)
	}

	db.MarkPageAsIndexed(5, "https://example.com/1")
	db.MarkPageAsIndexed(10, "https://example.com/2")
	db.MarkPageAsIndexed(3, "https://example.com/3")

	lastID, err = db.GetLastIndexedPageID()
	if err != nil {
		t.Fatalf("Failed to get last indexed page ID: %v", err)
	}
	if lastID != 10 {
		t.Errorf("Expected last ID to be 10, got %d", lastID)
	}
}

func TestMetadata(t *testing.T) {
	dbPath := "/tmp/test_index_6.db"
	defer os.Remove(dbPath)

	db, err := storage.NewIndexDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create index DB: %v", err)
	}
	defer db.Close()

	value, err := db.GetMetadata("total_documents")
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}
	if value != "0" {
		t.Errorf("Expected total_documents to be '0', got %q", value)
	}

	err = db.SetMetadata("total_documents", "100")
	if err != nil {
		t.Fatalf("Failed to set metadata: %v", err)
	}

	value, err = db.GetMetadata("total_documents")
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}
	if value != "100" {
		t.Errorf("Expected total_documents to be '100', got %q", value)
	}
}
