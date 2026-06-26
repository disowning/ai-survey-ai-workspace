CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE knowledge_chunks
ADD COLUMN IF NOT EXISTS embedding vector(1536);

ALTER TABLE knowledge_chunks
ADD COLUMN IF NOT EXISTS search_vector tsvector
GENERATED ALWAYS AS (to_tsvector('simple', coalesce(chunk_text, ''))) STORED;

CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_search_vector
ON knowledge_chunks USING GIN (search_vector);

CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_embedding_cosine
ON knowledge_chunks USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100)
WHERE embedding IS NOT NULL;
