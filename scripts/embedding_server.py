#!/usr/bin/env python3
"""
Simple embedding server using sentence-transformers.

Run with:
    pip install sentence-transformers flask
    python scripts/embedding_server.py

Or with gunicorn for production:
    pip install gunicorn
    gunicorn -w 4 -b 0.0.0.0:8081 scripts.embedding_server:app
"""

import os
from flask import Flask, request, jsonify
from sentence_transformers import SentenceTransformer

app = Flask(__name__)

# Load model (downloads on first run, then cached)
MODEL_NAME = os.environ.get("EMBEDDING_MODEL", "all-MiniLM-L6-v2")
print(f"Loading model: {MODEL_NAME}")
model = SentenceTransformer(MODEL_NAME)
print(f"Model loaded. Dimension: {model.get_sentence_embedding_dimension()}")


@app.route("/health", methods=["GET"])
def health():
    """Health check endpoint."""
    return jsonify({"status": "healthy", "model": MODEL_NAME})


@app.route("/embed", methods=["POST"])
def embed():
    """Generate embeddings for input texts."""
    data = request.get_json()

    if not data or "texts" not in data:
        return jsonify({"error": "Missing 'texts' field"}), 400

    texts = data["texts"]
    if not isinstance(texts, list):
        return jsonify({"error": "'texts' must be a list"}), 400

    if len(texts) == 0:
        return jsonify({"embeddings": []})

    if len(texts) > 100:
        return jsonify({"error": "Maximum 100 texts per request"}), 400

    try:
        embeddings = model.encode(texts)
        return jsonify({
            "embeddings": embeddings.tolist(),
            "dimension": model.get_sentence_embedding_dimension()
        })
    except Exception as e:
        return jsonify({"error": str(e)}), 500


@app.route("/similarity", methods=["POST"])
def similarity():
    """Calculate similarity between two texts."""
    data = request.get_json()

    if not data or "text1" not in data or "text2" not in data:
        return jsonify({"error": "Missing 'text1' or 'text2' field"}), 400

    try:
        embeddings = model.encode([data["text1"], data["text2"]])
        # Cosine similarity
        from numpy import dot
        from numpy.linalg import norm
        similarity = float(dot(embeddings[0], embeddings[1]) / (norm(embeddings[0]) * norm(embeddings[1])))
        return jsonify({"similarity": similarity})
    except Exception as e:
        return jsonify({"error": str(e)}), 500


if __name__ == "__main__":
    port = int(os.environ.get("PORT", 8081))
    debug = os.environ.get("DEBUG", "false").lower() == "true"
    print(f"Starting embedding server on port {port}")
    app.run(host="0.0.0.0", port=port, debug=debug)
