package config

import "os"

type Config struct {
	HTTPAddr         string
	DatabaseURL      string
	AIAPIKey         string
	AIBaseURL        string
	AIModel          string
	AIEmbeddingModel string
}

func Load() Config {
	return Config{
		HTTPAddr:         getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:      getenv("DATABASE_URL", "postgres://survey_ai:survey_ai@localhost:5432/survey_ai_workspace?sslmode=disable"),
		AIAPIKey:         getenv("AI_API_KEY", ""),
		AIBaseURL:        getenv("AI_BASE_URL", "https://api.openai.com/v1"),
		AIModel:          getenv("AI_MODEL", "gpt-4.1-mini"),
		AIEmbeddingModel: getenv("AI_EMBEDDING_MODEL", "text-embedding-3-small"),
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
