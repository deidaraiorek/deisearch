# Query Engine

A Go HTTP API that serves both classic TF-IDF keyword search and semantic vector search over the same crawled page corpus.

## Architecture

**Flow:** Frontend / Client -> Query Engine -> `/search` (keyword path) or `/semantic-search` (semantic path) -> `spider.db` metadata -> JSON response

**Components:**

- **HTTP Router**: Chi-based API router with logger, recoverer, and CORS middleware
- **Search Handler**: Processes keyword queries using text normalization, term lookup, TF-IDF ranking, and pagination
- **Semantic Search Handler**: Embeds query text, searches HNSW nearest neighbors, and enriches matches with page metadata
- **Index Reader**: Reads normalized terms and postings from `index.db`
- **Spider Reader**: Reads page title, description, URL, and content from `spider.db`
- **Embeddings Reader**: Loads stored document embeddings from `embeddings.db`
- **HNSW Index**: In-memory approximate nearest-neighbor graph built at startup for semantic search
- **Embedding Model Client**: HTTP client to `embedding-service/` for query embeddings

## Usage

```bash
go run main.go
```

The query engine starts on `http://localhost:8080`. Keyword search works as long as `index.db` and `spider.db` are available. Semantic search additionally requires a healthy embedding service and a populated `embeddings.db`.

## How It Works

**Keyword Search Path (`/search`):**

- Validates query and page parameters
- Normalizes query text with tokenization, stopword removal, and stemming
- Maps query terms to `term_id` values in `index.db`
- Ranks matching documents using TF-IDF-based SQL queries
- Fetches page metadata from `spider.db`
- Returns paginated JSON results

**Semantic Search Path (`/semantic-search`):**

- Calls the embedding service to embed the query text
- Loads the HNSW index built from `embeddings.db` at startup
- Searches nearest vector neighbors in memory
- Maps internal point IDs back to document IDs
- Fetches page metadata from `spider.db`
- Returns paginated JSON results

## API

**`GET /hello`:**

- Simple health-style endpoint

**`GET /search?q=...&page=...`:**

- Classic keyword search over `index.db`

**`GET /semantic-search?q=...&page=...`:**

- Semantic vector search over `embeddings.db`

## Data Sources

- **`index.db`**: Terms, postings, and TF-IDF scores for keyword retrieval
- **`spider.db`**: Page metadata and content snippets for response enrichment
- **`embeddings.db`**: Serialized 384-dimensional embeddings used to build the HNSW index
