package storage

const Schema = `
-- Terms dictionary: stores unique terms from all documents
CREATE TABLE IF NOT EXISTS terms (
    term_id INTEGER PRIMARY KEY AUTOINCREMENT,
    term TEXT UNIQUE NOT NULL,
    document_frequency INTEGER DEFAULT 0,
    idf REAL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_terms_term ON terms(term);

-- Postings list: inverted index mapping terms to documents
CREATE TABLE IF NOT EXISTS postings (
    term_id INTEGER NOT NULL,
    doc_id INTEGER NOT NULL,
    term_frequency INTEGER NOT NULL,
    tf REAL DEFAULT 0,
    tfidf REAL DEFAULT 0,
    PRIMARY KEY (term_id, doc_id),
    FOREIGN KEY (term_id) REFERENCES terms(term_id),
    FOREIGN KEY (doc_id) REFERENCES indexed_pages(doc_id)
);
CREATE INDEX IF NOT EXISTS idx_postings_term ON postings(term_id);
CREATE INDEX IF NOT EXISTS idx_postings_doc ON postings(doc_id);
-- Composite index for fast ranked search queries
CREATE INDEX IF NOT EXISTS idx_postings_term_tfidf ON postings(term_id, tfidf DESC, doc_id);

-- Document statistics: metadata for TF-IDF normalization
CREATE TABLE IF NOT EXISTS doc_stats (
    doc_id INTEGER PRIMARY KEY,
    doc_length INTEGER NOT NULL,      -- total number of terms in document
    unique_terms INTEGER NOT NULL,    -- number of unique terms
    indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (doc_id) REFERENCES indexed_pages(doc_id)
);

-- Track which pages from the spider DB have been indexed
-- This prevents reprocessing and allows resumable indexing
CREATE TABLE IF NOT EXISTS indexed_pages (
    doc_id INTEGER PRIMARY KEY,       -- references pages.id from spider DB
    source_url TEXT NOT NULL,         -- the original URL (for debugging)
    indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_indexed_pages_url ON indexed_pages(source_url);

-- Index metadata: track global indexing state
CREATE TABLE IF NOT EXISTS index_metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Initialize metadata with default values
INSERT OR IGNORE INTO index_metadata (key, value) VALUES
    ('total_documents', '0'),
    ('last_indexed_page_id', '0'),
    ('index_version', '1'),
    ('indexing_complete', 'false');
`
