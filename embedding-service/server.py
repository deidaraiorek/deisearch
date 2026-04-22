#!/usr/bin/env python3
import sys
from flask import Flask, request, jsonify
from sentence_transformers import SentenceTransformer

app = Flask(__name__)

print("Loading all-MiniLM-L6-v2 model...", file=sys.stderr)
model = SentenceTransformer('sentence-transformers/all-MiniLM-L6-v2')
print("Model loaded successfully!", file=sys.stderr)

@app.route('/embed', methods=['POST'])
def embed():
    data = request.get_json()

    if 'text' in data:
        text = data['text']
        embedding = model.encode(text, normalize_embeddings=True)
        return jsonify({'embedding': embedding.tolist()})

    elif 'texts' in data:
        texts = data['texts']
        embeddings = model.encode(texts, normalize_embeddings=True, show_progress_bar=False)
        return jsonify({'embeddings': embeddings.tolist()})

    return jsonify({'error': 'Missing text or texts field'}), 400

@app.route('/health', methods=['GET'])
def health():
    return jsonify({'status': 'ok', 'model': 'all-MiniLM-L6-v2', 'dimensions': 384})

if __name__ == '__main__':
    app.run(host='127.0.0.1', port=5000, debug=False)
