package server

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"survey-ai-workspace/internal/ai"
)

type Server struct {
	db       *sql.DB
	aiClient *ai.Client
}

func New(db *sql.DB, aiClient *ai.Client) *gin.Engine {
	s := &Server{db: db, aiClient: aiClient}
	r := gin.Default()
	r.Use(corsMiddleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		api.POST("/profiles/bind", s.bindProfile)
		api.GET("/profiles", s.listProfiles)
		api.GET("/profiles/:id", s.getProfile)
		api.PUT("/profiles/:id", s.updateProfile)

		api.POST("/sites/detect", s.detectSite)
		api.GET("/sites", s.listSites)
		api.POST("/sites", s.createSite)
		api.GET("/sites/:site_key", s.getSite)
		api.PUT("/sites/:site_key", s.updateSite)

		api.POST("/profile-sites", s.createProfileSite)
		api.GET("/profile-sites", s.listProfileSites)
		api.GET("/profile-sites/:id", s.getProfileSite)
		api.PUT("/profile-sites/:id", s.updateProfileSite)

		api.POST("/surveys", s.createSurvey)
		api.GET("/surveys", s.listSurveys)
		api.GET("/surveys/:id", s.getSurvey)
		api.PUT("/surveys/:id", s.updateSurvey)

		api.POST("/page-snapshots", s.createPageSnapshot)
		api.GET("/page-snapshots", s.listPageSnapshots)
		api.GET("/page-snapshots/:id", s.getPageSnapshot)

		api.POST("/notes", s.createNote)
		api.GET("/notes", s.listNotes)
		api.GET("/notes/:id", s.getNote)
		api.PUT("/notes/:id", s.updateNote)
		api.DELETE("/notes/:id", s.deleteNote)

		api.POST("/ai/translate", s.translateText)
		api.GET("/translations", s.listTranslations)
		api.GET("/translations/:id", s.getTranslation)

		api.POST("/ai/chat", s.chatWithAI)
		api.GET("/ai/conversations", s.listAIConversations)
		api.GET("/ai/conversations/:id", s.getAIConversation)

		api.POST("/knowledge", s.createKnowledgeChunk)
		api.GET("/knowledge", s.listKnowledgeChunks)
		api.GET("/knowledge/search", s.searchKnowledgeChunks)
		api.DELETE("/knowledge/:id", s.deleteKnowledgeChunk)

		api.GET("/export/markdown", s.exportMarkdown)
		api.GET("/export/notes.csv", s.exportNotesCSV)
		api.GET("/export/daily-report", s.exportDailyReport)

		api.GET("/system/status", s.systemStatus)
	}

	return r
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
