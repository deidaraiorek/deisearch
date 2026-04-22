package storage

const Schema = `
CREATE TABLE IF NOT EXISTS embeddings (
	doc_id INTEGER PRIMARY KEY,
	embedding BLOB NOT NULL,
	indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS indexed_pages (
	doc_id INTEGER PRIMARY KEY,
	source_url TEXT NOT NULL,
	indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_indexed_pages_url ON indexed_pages(source_url);

CREATE TABLE IF NOT EXISTS embedding_metadata (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO embedding_metadata (key, value) VALUES
	('model_name', 'all-MiniLM-L6-v2'),
	('embedding_dim', '384'),
	('last_indexed_page_id', '0'),
	('total_embeddings', '0'),
	('normalization', 'l2'),
	('indexing_complete', 'false');
`
