from vertexai import rag
from vertexai.generative_models import GenerativeModel, Tool
import vertexai

# TODO(developer): 以下をご自身の環境に合わせて更新してください。
PROJECT_ID = "your-project-id"  # あなたのプロジェクトID
LOCATION = "your-location"  # あなたのロケーション
display_name = "test_corpus"    # RAG コーパスの表示名
paths = ["https://drive.google.com/file/d/123", "gs://my_bucket/my_files_dir"] # インポートするファイルパスのリスト (Google StorageまたはGoogle Drive のリンクに対応)


def run_rag_quickstart():
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
    if PROJECT_ID == "your-project-id" or LOCATION == "your-location":
        print("⚠️ 警告: PROJECT_ID や LOCATION などのパラメータがデフォルト値のままです。")
        print("スクリプト内のTODOセクションを自身の環境に合わせて書き換えてから再実行してください。")
        exit(1)
    
    run_rag_quickstart()
