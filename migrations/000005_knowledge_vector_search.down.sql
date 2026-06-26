DROP INDEX IF EXISTS idx_knowledge_chunks_embedding_cosine;
DROP INDEX IF EXISTS idx_knowledge_chunks_search_vector;

ALTER TABLE knowledge_chunks
DROP COLUMN IF EXISTS search_vector;

ALTER TABLE knowledge_chunks
DROP COLUMN IF EXISTS embedding;
