package storage

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"

	_ "github.com/mattn/go-sqlite3"
)

type EmbeddingsReader struct {
	db *sql.DB
}

func NewEmbeddingsReader(dbPath string) (*EmbeddingsReader, error) {
	db, err := sql.Open("sqlite3", dbPath+"?cache=shared&mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open embeddings database: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	return &EmbeddingsReader{db: db}, nil
}

func (er *EmbeddingsReader) GetAllEmbeddings() ([][]float32, []int, error) {
	rows, err := er.db.Query("SELECT doc_id, embedding FROM embeddings ORDER BY doc_id")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query embeddings: %w", err)
	}
	defer rows.Close()

	var embeddings [][]float32
	var docIDs []int

	for rows.Next() {
		var docID int
		var embBytes []byte

		if err := rows.Scan(&docID, &embBytes); err != nil {
			return nil, nil, fmt.Errorf("failed to scan row: %w", err)
		}

		emb, err := deserializeEmbedding(embBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to deserialize embedding for doc %d: %w", docID, err)
		}

		docIDs = append(docIDs, docID)
		embeddings = append(embeddings, emb)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return embeddings, docIDs, nil
}

func (er *EmbeddingsReader) GetEmbeddingByDocID(docID int) ([]float32, error) {
	var embBytes []byte
	err := er.db.QueryRow("SELECT embedding FROM embeddings WHERE doc_id = ?", docID).Scan(&embBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding for doc %d: %w", docID, err)
	}

	return deserializeEmbedding(embBytes)
}

func (er *EmbeddingsReader) Close() error {
	return er.db.Close()
}

func deserializeEmbedding(data []byte) ([]float32, error) {
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("invalid embedding size: %d bytes", len(data))
	}

	numFloats := len(data) / 4
	vec := make([]float32, numFloats)

	for i := 0; i < numFloats; i++ {
		bits := binary.LittleEndian.Uint32(data[i*4:])
		vec[i] = math.Float32frombits(bits)
	}

	return vec, nil
}
