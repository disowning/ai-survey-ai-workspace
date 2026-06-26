package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"survey-ai-workspace/internal/ai"
	"survey-ai-workspace/internal/config"
	"survey-ai-workspace/internal/database"
	"survey-ai-workspace/internal/server"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	aiClient := ai.NewClient(cfg.AIAPIKey, cfg.AIBaseURL, cfg.AIModel, cfg.AIEmbeddingModel)
	router := server.New(db, aiClient)

	go func() {
		if err := router.Run(cfg.HTTPAddr); err != nil {
			log.Fatalf("run server: %v", err)
		}
	}()

	log.Printf("survey ai workspace api listening on %s", cfg.HTTPAddr)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutdown requested")
}
