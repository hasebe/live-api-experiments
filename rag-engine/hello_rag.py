from vertexai import rag
from vertexai.generative_models import GenerativeModel, Tool
import vertexai
import os

# Configuration via environment variables
PROJECT_ID = os.environ.get("GOOGLE_CLOUD_PROJECT")
LOCATION = os.environ.get("RAG_LOCATION", os.environ.get("GOOGLE_CLOUD_LOCATION", "us-central1"))
display_name = os.environ.get("RAG_CORPUS_NAME", "test_corpus")

# Comma-separated list of paths
paths_str = os.environ.get("RAG_IMPORT_PATHS", "")
paths = [p.strip() for p in paths_str.split(",")] if paths_str else []

def run_rag_quickstart():
    if not PROJECT_ID:
        raise ValueError("GOOGLE_CLOUD_PROJECT environment variable is not set.")
    if not paths:
        print("Warning: RAG_IMPORT_PATHS is not set. No files will be imported.")

    print(f"Initializing Vertex AI API for project: {PROJECT_ID} (location: {LOCATION})")
    # Initialize Vertex AI API once per session
    vertexai.init(project=PROJECT_ID, location=LOCATION)

    print("\n--- Creating RagCorpus ---")
    # Configure embedding model, for example "text-embedding-005".
    embedding_model_config = rag.RagEmbeddingModelConfig(
        vertex_prediction_endpoint=rag.VertexPredictionEndpoint(
            publisher_model="publishers/google/models/text-multilingual-embedding-002"
        )
    )

    rag_corpus = rag.create_corpus(
        display_name=display_name,
        backend_config=rag.RagVectorDbConfig(
            rag_embedding_model_config=embedding_model_config
        ),
    )
    print(f"Created RAG corpus: {rag_corpus.name}")

    print("\n--- Importing Files to the RagCorpus ---")
    rag.import_files(
        rag_corpus.name,
        paths,
        # Optional
        transformation_config=rag.TransformationConfig(
            chunking_config=rag.ChunkingConfig(
                chunk_size=512,
                chunk_overlap=100,
            ),
        ),
        max_embedding_requests_per_min=1000,  # Optional
    )
    print("Files imported successfully.")

    print("\n--- Direct context retrieval ---")
    rag_retrieval_config = rag.RagRetrievalConfig(
        top_k=3,  # Optional
        filter=rag.Filter(vector_distance_threshold=0.5),  # Optional
    )

    query = "ゼロトラストとはなんですか？"
    print(f"Querying: '{query}'")
    response = rag.retrieval_query(
        rag_resources=[
            rag.RagResource(
                rag_corpus=rag_corpus.name,
                # Optional: supply IDs from `rag.list_files()`.
                # rag_file_ids=["rag-file-1", "rag-file-2", ...],
            )
        ],
        text=query,
        rag_retrieval_config=rag_retrieval_config,
    )
    print("Retrieval Query Response:")
    print(response)

    print("\n--- Enhance generation ---")
    # Create a RAG retrieval tool
    rag_retrieval_tool = Tool.from_retrieval(
        retrieval=rag.Retrieval(
            source=rag.VertexRagStore(
                rag_resources=[
                    rag.RagResource(
                        rag_corpus=rag_corpus.name,
                        # Currently only 1 corpus is allowed.
                        # Optional: supply IDs from `rag.list_files()`.
                        # rag_file_ids=["rag-file-1", "rag-file-2", ...],
                    )
                ],
                rag_retrieval_config=rag_retrieval_config,
            ),
        )
    )

    # Create a Gemini model instance
    rag_model = GenerativeModel(
        model_name="gemini-2.5-flash", tools=[rag_retrieval_tool]
    )

    # Generate response
    response_gen = rag_model.generate_content(query)
    print("\nGeneration Response:")
    print(response_gen.text)


if __name__ == "__main__":
    try:
        run_rag_quickstart()
    except Exception as e:
        print(f"Error: {e}")
        exit(1)
