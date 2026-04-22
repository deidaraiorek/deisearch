# Embedding Service

A Python Flask microservice that generates normalized 384-dimensional sentence embeddings using `all-MiniLM-L6-v2` for both semantic indexing and semantic search.

## Architecture

**Flow:** Client (Semantic Indexer / Query Engine) -> Flask API -> SentenceTransformer Model -> Normalized Embedding Response

**Components:**

- **Flask API**: Exposes `/health` and `/embed` endpoints
- **SentenceTransformer Model**: Loads `sentence-transformers/all-MiniLM-L6-v2` once at startup
- **Single Text Path**: Accepts `{"text": "..."}` and returns one embedding
- **Batch Path**: Accepts `{"texts": ["...", "..."]}` and returns multiple embeddings
- **Normalization**: Uses `normalize_embeddings=True` so dot product aligns with cosine similarity

## Setup

```bash
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

## Usage

```bash
python server.py
```

The service starts on `http://127.0.0.1:5000`, loads the model once, and stays available for both batch indexing requests and per-query semantic search requests.

## How It Works

**Embedding Strategy:**

- Loads `all-MiniLM-L6-v2` at process start
- Generates 384-dimensional embeddings
- Returns normalized vectors for stable similarity search
- Supports both single-query inference and batch indexing inference

**Request Modes:**

- **Single text**: Used by `query-engine/` for `/semantic-search`
- **Batch texts**: Used by `semantic-indexer/` to process pages in batches

## API

**`GET /health`:**

- Returns status, model name, and embedding dimensions

```bash
curl http://127.0.0.1:5000/health
```

**`POST /embed` (single text):**

```bash
curl -X POST http://127.0.0.1:5000/embed \
  -H "Content-Type: application/json" \
  -d '{"text": "machine learning"}'
```

**`POST /embed` (batch texts):**

```bash
curl -X POST http://127.0.0.1:5000/embed \
  -H "Content-Type: application/json" \
  -d '{"texts": ["machine learning", "artificial intelligence"]}'
```
