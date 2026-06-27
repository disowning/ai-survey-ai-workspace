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
	auth     AuthConfig
}

type AuthConfig struct {
	Username    string
	Password    string
	TokenSecret string
}

func New(db *sql.DB, aiClient *ai.Client, auth AuthConfig) *gin.Engine {
	s := &Server{db: db, aiClient: aiClient, auth: auth}
	r := gin.Default()
	r.Use(corsMiddleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		api.POST("/auth/login", s.login)
		api.GET("/system/status", s.systemStatus)

		protected := api.Group("")
		protected.Use(s.authMiddleware())
		{
			protected.POST("/profiles/bind", s.bindProfile)
			protected.GET("/profiles", s.listProfiles)
			protected.GET("/profiles/:id", s.getProfile)
			protected.PUT("/profiles/:id", s.updateProfile)

			protected.POST("/sites/detect", s.detectSite)
			protected.GET("/sites", s.listSites)
			protected.POST("/sites", s.createSite)
			protected.GET("/sites/:site_key", s.getSite)
			protected.PUT("/sites/:site_key", s.updateSite)

			protected.POST("/profile-sites", s.createProfileSite)
			protected.GET("/profile-sites", s.listProfileSites)
			protected.GET("/profile-sites/:id", s.getProfileSite)
			protected.PUT("/profile-sites/:id", s.updateProfileSite)

			protected.POST("/survey-sessions/ensure", s.ensureSurvey)
			protected.POST("/surveys", s.createSurvey)
			protected.GET("/surveys", s.listSurveys)
			protected.GET("/surveys/:id", s.getSurvey)
			protected.PUT("/surveys/:id", s.updateSurvey)

			protected.POST("/page-snapshots", s.createPageSnapshot)
			protected.GET("/page-snapshots", s.listPageSnapshots)
			protected.GET("/page-snapshots/:id", s.getPageSnapshot)

			protected.POST("/notes", s.createNote)
			protected.GET("/notes", s.listNotes)
			protected.GET("/notes/:id", s.getNote)
			protected.PUT("/notes/:id", s.updateNote)
			protected.DELETE("/notes/:id", s.deleteNote)

			protected.POST("/ai/translate", s.translateText)
			protected.GET("/translations", s.listTranslations)
			protected.GET("/translations/:id", s.getTranslation)

			protected.POST("/ai/chat", s.chatWithAI)
			protected.GET("/ai/conversations", s.listAIConversations)
			protected.GET("/ai/conversations/:id", s.getAIConversation)

			protected.POST("/knowledge", s.createKnowledgeChunk)
			protected.GET("/knowledge", s.listKnowledgeChunks)
			protected.GET("/knowledge/search", s.searchKnowledgeChunks)
			protected.DELETE("/knowledge/:id", s.deleteKnowledgeChunk)

			protected.POST("/personas", s.createSitePersona)
			protected.GET("/personas", s.listSitePersonas)
			protected.GET("/personas/context", s.sitePersonaContext)
			protected.PUT("/personas/:id", s.updateSitePersona)
			protected.DELETE("/personas/:id", s.deleteSitePersona)

			protected.GET("/export/markdown", s.exportMarkdown)
			protected.GET("/export/notes.csv", s.exportNotesCSV)
			protected.GET("/export/daily-report", s.exportDailyReport)
		}
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
