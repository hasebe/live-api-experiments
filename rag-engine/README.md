# Vertex AI RAG Engine Setup Scripts

This directory contains Python scripts for initializing and managing the Vertex AI RAG Engine.

## Prerequisites
- Built with `uv`. Use `uv run` to execute the scripts safely within the isolated environment.

## `hello_rag.py`

This script initializes a Vertex AI RAG Corpus, imports specified files, and runs a sample query and generative model generation using the `gemini-2.5-flash` model.

### Configuration

Instead of modifying the script directly, configure the execution via the following environment variables:

- `GOOGLE_CLOUD_PROJECT` (Required): Your Google Cloud Project ID.
- `RAG_LOCATION` (Optional): The region for the RAG endpoint. Defaults to `GOOGLE_CLOUD_LOCATION`, or `us-central1`.
- `RAG_CORPUS_NAME` (Optional): Display name for the RAG Corpus. Defaults to `test_corpus`.
- `RAG_IMPORT_PATHS` (Optional): A comma-separated list of paths to import. Supports Google Cloud Storage (`gs://...`) or Google Drive (`https://drive.google.com/...`) links.

### Usage Example

```bash
cd rag-engine
export GOOGLE_CLOUD_PROJECT="your-project-id"
export RAG_LOCATION="europe-west3"
export RAG_CORPUS_NAME="my_knowledge_base"
export RAG_IMPORT_PATHS="gs://my_bucket/zero_trust.pdf,https://drive.google.com/file/d/123"

uv run hello_rag.py
```
