package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"survey-ai-workspace/internal/ai"
	"survey-ai-workspace/internal/models"
)

type scanner interface {
	Scan(dest ...any) error
}

const requestTimeout = 10 * time.Second
const aiRequestTimeout = 90 * time.Second
const knowledgeEmbeddingDimensions = 1536

func requestContext(c *gin.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), requestTimeout)
}

func aiRequestContext(c *gin.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), aiRequestTimeout)
}

func badRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": message})
}

func notFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
}

func serverError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func handleDBError(c *gin.Context, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		notFound(c)
		return
	}
	serverError(c, err)
}

func writeDeleteResult(c *gin.Context, result sql.Result) bool {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		handleDBError(c, err)
		return false
	}
	if rowsAffected == 0 {
		notFound(c)
		return false
	}
	return true
}

func (s *Server) systemStatus(c *gin.Context) {
	ctx, cancel := requestContext(c)
	defer cancel()

	databaseOK := s.db.PingContext(ctx) == nil
	c.JSON(http.StatusOK, gin.H{
		"status":                "ok",
		"auth_required":         s.authEnabled(),
		"database_ok":           databaseOK,
		"ai_configured":         s.aiClient.Configured(),
		"embedding_configured":  s.aiClient.EmbeddingsConfigured(),
		"ai_model":              s.aiClient.ChatModel(),
		"ai_embedding_model":    s.aiClient.EmbeddingModel(),
		"knowledge_vector_size": knowledgeEmbeddingDimensions,
	})
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		badRequest(c, "invalid "+name)
		return 0, false
	}
	return id, true
}

func optionalInt64Query(c *gin.Context, key string) any {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return nil
	}
	return parsed
}

func optionalStringQuery(c *gin.Context, key string) any {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return nil
	}
	return value
}

func limitOffset(c *gin.Context) (int, int) {
	limit := 50
	offset := 0

	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 200 {
		limit = 200
	}

	if raw := strings.TrimSpace(c.Query("offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}

func ptrValue[T any](value *T) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Int64
}

func scanProfile(s scanner) (models.Profile, error) {
	var row models.Profile
	var profileName, remark sql.NullString
	err := s.Scan(&row.ID, &row.ProfileKey, &profileName, &remark, &row.Status, &row.CreatedAt, &row.UpdatedAt)
	row.ProfileName = nullStringPtr(profileName)
	row.Remark = nullStringPtr(remark)
	return row, err
}

func scanSurveySite(s scanner) (models.SurveySite, error) {
	var row models.SurveySite
	var siteName, domain sql.NullString
	err := s.Scan(&row.ID, &row.SiteKey, &siteName, &domain, &row.CreatedAt, &row.UpdatedAt)
	row.SiteName = nullStringPtr(siteName)
	row.Domain = nullStringPtr(domain)
	return row, err
}

func scanProfileSite(s scanner) (models.ProfileSite, error) {
	var row models.ProfileSite
	var email, remark sql.NullString
	err := s.Scan(&row.ID, &row.ProfileID, &row.SiteKey, &email, &remark, &row.CreatedAt, &row.UpdatedAt)
	row.SiteAccountEmail = nullStringPtr(email)
	row.SiteRemark = nullStringPtr(remark)
	return row, err
}

func scanSurvey(s scanner) (models.Survey, error) {
	var row models.Survey
	var title, surveyURL sql.NullString
	err := s.Scan(&row.ID, &row.ProfileID, &row.SiteKey, &title, &surveyURL, &row.Status, &row.CreatedAt, &row.UpdatedAt)
	row.SurveyTitle = nullStringPtr(title)
	row.SurveyURL = nullStringPtr(surveyURL)
	return row, err
}

func scanPageSnapshot(s scanner) (models.PageSnapshot, error) {
	var row models.PageSnapshot
	var surveyID sql.NullInt64
	var pageURL, pageTitle, question, options, pageText, language sql.NullString
	err := s.Scan(
		&row.ID,
		&row.ProfileID,
		&row.SiteKey,
		&surveyID,
		&pageURL,
		&pageTitle,
		&question,
		&options,
		&pageText,
		&language,
		&row.CreatedAt,
	)
	row.SurveyID = nullInt64Ptr(surveyID)
	row.URL = nullStringPtr(pageURL)
	row.PageTitle = nullStringPtr(pageTitle)
	row.QuestionText = nullStringPtr(question)
	row.OptionsText = nullStringPtr(options)
	row.PageText = nullStringPtr(pageText)
	row.Language = nullStringPtr(language)
	return row, err
}

func scanNote(s scanner) (models.Note, error) {
	var row models.Note
	var surveyID, pageSnapshotID sql.NullInt64
	err := s.Scan(&row.ID, &row.ProfileID, &row.SiteKey, &surveyID, &pageSnapshotID, &row.NoteText, &row.CreatedAt, &row.UpdatedAt)
	row.SurveyID = nullInt64Ptr(surveyID)
	row.PageSnapshotID = nullInt64Ptr(pageSnapshotID)
	return row, err
}

func scanTranslation(s scanner) (models.Translation, error) {
	var row models.Translation
	var sourceLang sql.NullString
	err := s.Scan(
		&row.ID,
		&row.ProfileID,
		&row.SiteKey,
		&row.SourceText,
		&row.TranslatedText,
		&sourceLang,
		&row.TargetLang,
		&row.CreatedAt,
	)
	row.SourceLang = nullStringPtr(sourceLang)
	return row, err
}

func scanAIConversation(s scanner) (models.AIConversation, error) {
	var row models.AIConversation
	var surveyID sql.NullInt64
	err := s.Scan(
		&row.ID,
		&row.ProfileID,
		&row.SiteKey,
		&surveyID,
		&row.UserMessage,
		&row.AIMessage,
		&row.CreatedAt,
	)
	row.SurveyID = nullInt64Ptr(surveyID)
	return row, err
}

func scanKnowledgeChunk(s scanner) (models.KnowledgeChunk, error) {
	var row models.KnowledgeChunk
	var surveyID, sourceID sql.NullInt64
	err := s.Scan(
		&row.ID,
		&row.ProfileID,
		&row.SiteKey,
		&surveyID,
		&row.Scope,
		&row.SourceType,
		&sourceID,
		&row.ChunkText,
		&row.CreatedAt,
	)
	row.SurveyID = nullInt64Ptr(surveyID)
	row.SourceID = nullInt64Ptr(sourceID)
	return row, err
}

func scanSitePersona(s scanner) (models.SitePersona, error) {
	var row models.SitePersona
	var sourceID sql.NullInt64
	err := s.Scan(
		&row.ID,
		&row.ProfileID,
		&row.SiteKey,
		&row.Category,
		&row.PersonaKey,
		&row.PersonaValue,
		&row.Confidence,
		&row.SourceType,
		&sourceID,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	row.SourceID = nullInt64Ptr(sourceID)
	return row, err
}

type profileRequest struct {
	ProfileKey  string  `json:"profile_key"`
	ProfileName *string `json:"profile_name"`
	Remark      *string `json:"remark"`
	Status      *string `json:"status"`
}

func (s *Server) bindProfile(c *gin.Context) {
	var req profileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}
	req.ProfileKey = strings.TrimSpace(req.ProfileKey)
	if req.ProfileKey == "" {
		badRequest(c, "profile_key is required")
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanProfile(s.db.QueryRowContext(ctx, `
		INSERT INTO profiles (profile_key, profile_name, remark)
		VALUES ($1, $2, $3)
		ON CONFLICT (profile_key) DO UPDATE SET
			profile_name = COALESCE(EXCLUDED.profile_name, profiles.profile_name),
			remark = COALESCE(EXCLUDED.remark, profiles.remark),
			updated_at = NOW()
		RETURNING id, profile_key, profile_name, remark, status, created_at, updated_at
	`, req.ProfileKey, ptrValue(req.ProfileName), ptrValue(req.Remark)))
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, row)
}

func (s *Server) listProfiles(c *gin.Context) {
	limit, offset := limitOffset(c)
	ctx, cancel := requestContext(c)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, profile_key, profile_name, remark, status, created_at, updated_at
		FROM profiles
		WHERE ($1::text IS NULL OR status = $1)
		ORDER BY id DESC
		LIMIT $2 OFFSET $3
	`, optionalStringQuery(c, "status"), limit, offset)
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer rows.Close()

	items := make([]models.Profile, 0)
	for rows.Next() {
		item, err := scanProfile(rows)
		if err != nil {
			handleDBError(c, err)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) getProfile(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanProfile(s.db.QueryRowContext(ctx, `
		SELECT id, profile_key, profile_name, remark, status, created_at, updated_at
		FROM profiles
		WHERE id = $1
	`, id))
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, row)
}

func (s *Server) updateProfile(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req profileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanProfile(s.db.QueryRowContext(ctx, `
		UPDATE profiles
		SET profile_name = COALESCE($2, profile_name),
			remark = COALESCE($3, remark),
			status = COALESCE($4, status),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, profile_key, profile_name, remark, status, created_at, updated_at
	`, id, ptrValue(req.ProfileName), ptrValue(req.Remark), ptrValue(req.Status)))
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, row)
}

type siteRequest struct {
	SiteKey  string  `json:"site_key"`
	SiteName *string `json:"site_name"`
	Domain   *string `json:"domain"`
}

func (s *Server) createSite(c *gin.Context) {
	var req siteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}
	req.SiteKey = strings.TrimSpace(req.SiteKey)
	if req.SiteKey == "" {
		badRequest(c, "site_key is required")
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanSurveySite(s.db.QueryRowContext(ctx, `
		INSERT INTO survey_sites (site_key, site_name, domain)
		VALUES ($1, $2, $3)
		ON CONFLICT (site_key) DO UPDATE SET
			site_name = COALESCE(EXCLUDED.site_name, survey_sites.site_name),
			domain = COALESCE(EXCLUDED.domain, survey_sites.domain),
			updated_at = NOW()
		RETURNING id, site_key, site_name, domain, created_at, updated_at
	`, req.SiteKey, ptrValue(req.SiteName), ptrValue(req.Domain)))
	if err != nil {
		handleDBError(c, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

func (s *Server) listSites(c *gin.Context) {
	limit, offset := limitOffset(c)
	ctx, cancel := requestContext(c)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, site_key, site_name, domain, created_at, updated_at
		FROM survey_sites
		WHERE ($1::text IS NULL OR site_key = $1)
			AND ($2::text IS NULL OR domain = $2)
		ORDER BY id DESC
		LIMIT $3 OFFSET $4
	`, optionalStringQuery(c, "site_key"), optionalStringQuery(c, "domain"), limit, offset)
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer rows.Close()

	items := make([]models.SurveySite, 0)
	for rows.Next() {
		item, err := scanSurveySite(rows)
		if err != nil {
			handleDBError(c, err)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) getSite(c *gin.Context) {
	siteKey := strings.TrimSpace(c.Param("site_key"))
	if siteKey == "" {
		badRequest(c, "site_key is required")
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanSurveySite(s.db.QueryRowContext(ctx, `
		SELECT id, site_key, site_name, domain, created_at, updated_at
		FROM survey_sites
		WHERE site_key = $1
	`, siteKey))
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, row)
}

func (s *Server) updateSite(c *gin.Context) {
	siteKey := strings.TrimSpace(c.Param("site_key"))
	if siteKey == "" {
		badRequest(c, "site_key is required")
		return
	}

	var req siteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanSurveySite(s.db.QueryRowContext(ctx, `
		UPDATE survey_sites
		SET site_name = COALESCE($2, site_name),
			domain = COALESCE($3, domain),
			updated_at = NOW()
		WHERE site_key = $1
		RETURNING id, site_key, site_name, domain, created_at, updated_at
	`, siteKey, ptrValue(req.SiteName), ptrValue(req.Domain)))
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, row)
}

type detectSiteRequest struct {
	URL string `json:"url"`
}

func (s *Server) detectSite(c *gin.Context) {
	var req detectSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}

	host, err := normalizedHost(req.URL)
	if err != nil {
		badRequest(c, "url is invalid")
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanSurveySite(s.db.QueryRowContext(ctx, `
		SELECT id, site_key, site_name, domain, created_at, updated_at
		FROM survey_sites
		WHERE LOWER(domain) = LOWER($1)
			OR LOWER($1) LIKE '%.' || LOWER(domain)
		ORDER BY LENGTH(domain) DESC
		LIMIT 1
	`, host))
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"site": row, "detected": true, "persisted": true})
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		handleDBError(c, err)
		return
	}

	siteKey := siteKeyFromHost(host)
	siteName := strings.Title(strings.ReplaceAll(siteKey, "-", " "))
	c.JSON(http.StatusOK, gin.H{
		"site": gin.H{
			"site_key":  siteKey,
			"site_name": siteName,
			"domain":    host,
		},
		"detected":  true,
		"persisted": false,
	})
}

func normalizedHost(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("empty url")
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	host := strings.ToLower(parsed.Hostname())
	host = strings.TrimPrefix(host, "www.")
	if host == "" {
		return "", fmt.Errorf("empty host")
	}
	return host, nil
}

func siteKeyFromHost(host string) string {
	parts := strings.Split(host, ".")
	candidate := parts[0]
	if len(parts) >= 2 {
		candidate = parts[len(parts)-2]
	}
	if len(parts) >= 3 && (candidate == "co" || candidate == "com" || candidate == "net" || candidate == "org") {
		candidate = parts[len(parts)-3]
	}

	re := regexp.MustCompile(`[^a-z0-9]+`)
	candidate = re.ReplaceAllString(strings.ToLower(candidate), "-")
	candidate = strings.Trim(candidate, "-")
	if candidate == "" {
		return "unknown-site"
	}
	return candidate
}

type profileSiteRequest struct {
	ProfileID        int64   `json:"profile_id"`
	SiteKey          string  `json:"site_key"`
	SiteAccountEmail *string `json:"site_account_email"`
	SiteRemark       *string `json:"site_remark"`
}

func (s *Server) createProfileSite(c *gin.Context) {
	var req profileSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}
	if req.ProfileID <= 0 || strings.TrimSpace(req.SiteKey) == "" {
		badRequest(c, "profile_id and site_key are required")
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanProfileSite(s.db.QueryRowContext(ctx, `
		INSERT INTO profile_sites (profile_id, site_key, site_account_email, site_remark)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (profile_id, site_key) DO UPDATE SET
			site_account_email = COALESCE(EXCLUDED.site_account_email, profile_sites.site_account_email),
			site_remark = COALESCE(EXCLUDED.site_remark, profile_sites.site_remark),
			updated_at = NOW()
		RETURNING id, profile_id, site_key, site_account_email, site_remark, created_at, updated_at
	`, req.ProfileID, strings.TrimSpace(req.SiteKey), ptrValue(req.SiteAccountEmail), ptrValue(req.SiteRemark)))
	if err != nil {
		handleDBError(c, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

func (s *Server) listProfileSites(c *gin.Context) {
	limit, offset := limitOffset(c)
	ctx, cancel := requestContext(c)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, profile_id, site_key, site_account_email, site_remark, created_at, updated_at
		FROM profile_sites
		WHERE ($1::bigint IS NULL OR profile_id = $1)
			AND ($2::text IS NULL OR site_key = $2)
		ORDER BY id DESC
		LIMIT $3 OFFSET $4
	`, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"), limit, offset)
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer rows.Close()

	items := make([]models.ProfileSite, 0)
	for rows.Next() {
		item, err := scanProfileSite(rows)
		if err != nil {
			handleDBError(c, err)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) getProfileSite(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanProfileSite(s.db.QueryRowContext(ctx, `
		SELECT id, profile_id, site_key, site_account_email, site_remark, created_at, updated_at
		FROM profile_sites
		WHERE id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key")))
	if err != nil {
		handleDBError(c, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

func (s *Server) updateProfileSite(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req profileSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanProfileSite(s.db.QueryRowContext(ctx, `
		UPDATE profile_sites
		SET site_account_email = COALESCE($2, site_account_email),
			site_remark = COALESCE($3, site_remark),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, profile_id, site_key, site_account_email, site_remark, created_at, updated_at
	`, id, ptrValue(req.SiteAccountEmail), ptrValue(req.SiteRemark)))
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, row)
}

type surveyRequest struct {
	ProfileID   int64   `json:"profile_id"`
	SiteKey     string  `json:"site_key"`
	SurveyTitle *string `json:"survey_title"`
	SurveyURL   *string `json:"survey_url"`
	Status      *string `json:"status"`
}

func (s *Server) createSurvey(c *gin.Context) {
	var req surveyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}
	if req.ProfileID <= 0 || strings.TrimSpace(req.SiteKey) == "" {
		badRequest(c, "profile_id and site_key are required")
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanSurvey(s.db.QueryRowContext(ctx, `
		INSERT INTO surveys (profile_id, site_key, survey_title, survey_url)
		VALUES ($1, $2, $3, $4)
		RETURNING id, profile_id, site_key, survey_title, survey_url, status, created_at, updated_at
	`, req.ProfileID, strings.TrimSpace(req.SiteKey), ptrValue(req.SurveyTitle), ptrValue(req.SurveyURL)))
	if err != nil {
		handleDBError(c, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

func (s *Server) ensureSurvey(c *gin.Context) {
	var req surveyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}
	req.SiteKey = strings.TrimSpace(req.SiteKey)
	if req.ProfileID <= 0 || req.SiteKey == "" {
		badRequest(c, "profile_id and site_key are required")
		return
	}
	title := strings.TrimSpace(ptrValueString(req.SurveyTitle))
	surveyURL := strings.TrimSpace(ptrValueString(req.SurveyURL))
	if title == "" {
		title = "临时问卷"
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	if surveyURL != "" {
		row, err := scanSurvey(s.db.QueryRowContext(ctx, `
			SELECT id, profile_id, site_key, survey_title, survey_url, status, created_at, updated_at
			FROM surveys
			WHERE profile_id = $1
				AND site_key = $2
				AND survey_url = $3
				AND status = 'active'
			ORDER BY id DESC
			LIMIT 1
		`, req.ProfileID, req.SiteKey, surveyURL))
		if err == nil {
			c.JSON(http.StatusOK, row)
			return
		}
		if err != sql.ErrNoRows {
			handleDBError(c, err)
			return
		}
	}

	row, err := scanSurvey(s.db.QueryRowContext(ctx, `
		INSERT INTO surveys (profile_id, site_key, survey_title, survey_url)
		VALUES ($1, $2, $3, $4)
		RETURNING id, profile_id, site_key, survey_title, survey_url, status, created_at, updated_at
	`, req.ProfileID, req.SiteKey, title, nullIfEmpty(surveyURL)))
	if err != nil {
		handleDBError(c, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

func (s *Server) listSurveys(c *gin.Context) {
	limit, offset := limitOffset(c)
	ctx, cancel := requestContext(c)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, profile_id, site_key, survey_title, survey_url, status, created_at, updated_at
		FROM surveys
		WHERE ($1::bigint IS NULL OR profile_id = $1)
			AND ($2::text IS NULL OR site_key = $2)
			AND ($3::text IS NULL OR status = $3)
		ORDER BY id DESC
		LIMIT $4 OFFSET $5
	`, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"), optionalStringQuery(c, "status"), limit, offset)
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer rows.Close()

	items := make([]models.Survey, 0)
	for rows.Next() {
		item, err := scanSurvey(rows)
		if err != nil {
			handleDBError(c, err)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) getSurvey(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanSurvey(s.db.QueryRowContext(ctx, `
		SELECT id, profile_id, site_key, survey_title, survey_url, status, created_at, updated_at
		FROM surveys
		WHERE id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key")))
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, row)
}

func (s *Server) updateSurvey(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req surveyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanSurvey(s.db.QueryRowContext(ctx, `
		UPDATE surveys
		SET survey_title = COALESCE($2, survey_title),
			survey_url = COALESCE($3, survey_url),
			status = COALESCE($4, status),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, profile_id, site_key, survey_title, survey_url, status, created_at, updated_at
	`, id, ptrValue(req.SurveyTitle), ptrValue(req.SurveyURL), ptrValue(req.Status)))
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, row)
}

func (s *Server) deleteSurvey(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	ctx, cancel := requestContext(c)
	defer cancel()

	result, err := s.db.ExecContext(ctx, `
		DELETE FROM surveys
		WHERE id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"))
	if err != nil {
		handleDBError(c, err)
		return
	}
	if !writeDeleteResult(c, result) {
		return
	}
	c.Status(http.StatusNoContent)
}

type pageSnapshotRequest struct {
	ProfileID    int64   `json:"profile_id"`
	SiteKey      string  `json:"site_key"`
	SurveyID     *int64  `json:"survey_id"`
	URL          *string `json:"url"`
	PageTitle    *string `json:"page_title"`
	QuestionText *string `json:"question_text"`
	OptionsText  *string `json:"options_text"`
	PageText     *string `json:"page_text"`
	Language     *string `json:"language"`
}

func (s *Server) createPageSnapshot(c *gin.Context) {
	var req pageSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}
	if req.ProfileID <= 0 || strings.TrimSpace(req.SiteKey) == "" {
		badRequest(c, "profile_id and site_key are required")
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanPageSnapshot(s.db.QueryRowContext(ctx, `
		INSERT INTO page_snapshots (
			profile_id, site_key, survey_id, url, page_title, question_text,
			options_text, page_text, language
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, profile_id, site_key, survey_id, url, page_title, question_text,
			options_text, page_text, language, created_at
	`, req.ProfileID, strings.TrimSpace(req.SiteKey), ptrValue(req.SurveyID), ptrValue(req.URL), ptrValue(req.PageTitle), ptrValue(req.QuestionText), ptrValue(req.OptionsText), ptrValue(req.PageText), ptrValue(req.Language)))
	if err != nil {
		handleDBError(c, err)
		return
	}
	s.createKnowledgeFromPageSnapshot(ctx, row)

	c.JSON(http.StatusCreated, row)
}

func (s *Server) listPageSnapshots(c *gin.Context) {
	limit, offset := limitOffset(c)
	ctx, cancel := requestContext(c)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, profile_id, site_key, survey_id, url, page_title, question_text,
			options_text, page_text, language, created_at
		FROM page_snapshots
		WHERE ($1::bigint IS NULL OR profile_id = $1)
			AND ($2::text IS NULL OR site_key = $2)
			AND ($3::bigint IS NULL OR survey_id = $3)
		ORDER BY id DESC
		LIMIT $4 OFFSET $5
	`, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"), optionalInt64Query(c, "survey_id"), limit, offset)
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer rows.Close()

	items := make([]models.PageSnapshot, 0)
	for rows.Next() {
		item, err := scanPageSnapshot(rows)
		if err != nil {
			handleDBError(c, err)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) getPageSnapshot(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanPageSnapshot(s.db.QueryRowContext(ctx, `
		SELECT id, profile_id, site_key, survey_id, url, page_title, question_text,
			options_text, page_text, language, created_at
		FROM page_snapshots
		WHERE id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
			AND ($4::bigint IS NULL OR survey_id = $4)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"), optionalInt64Query(c, "survey_id")))
	if err != nil {
		handleDBError(c, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

func (s *Server) deletePageSnapshot(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		DELETE FROM knowledge_chunks
		WHERE source_type = 'page_snapshot'
			AND source_id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"))
	if err != nil {
		handleDBError(c, err)
		return
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM page_snapshots
		WHERE id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"))
	if err != nil {
		handleDBError(c, err)
		return
	}
	if !writeDeleteResult(c, result) {
		return
	}
	if err := tx.Commit(); err != nil {
		handleDBError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type noteRequest struct {
	ProfileID      int64  `json:"profile_id"`
	SiteKey        string `json:"site_key"`
	SurveyID       *int64 `json:"survey_id"`
	PageSnapshotID *int64 `json:"page_snapshot_id"`
	NoteText       string `json:"note_text"`
}

func (s *Server) createNote(c *gin.Context) {
	var req noteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}
	req.SiteKey = strings.TrimSpace(req.SiteKey)
	req.NoteText = strings.TrimSpace(req.NoteText)
	if req.ProfileID <= 0 || req.SiteKey == "" || req.NoteText == "" {
		badRequest(c, "profile_id, site_key and note_text are required")
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanNote(s.db.QueryRowContext(ctx, `
		INSERT INTO notes (profile_id, site_key, survey_id, page_snapshot_id, note_text)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, profile_id, site_key, survey_id, page_snapshot_id, note_text, created_at, updated_at
	`, req.ProfileID, req.SiteKey, ptrValue(req.SurveyID), ptrValue(req.PageSnapshotID), req.NoteText))
	if err != nil {
		handleDBError(c, err)
		return
	}
	s.createKnowledgeFromNote(ctx, row)

	c.JSON(http.StatusCreated, row)
}

func (s *Server) listNotes(c *gin.Context) {
	limit, offset := limitOffset(c)
	ctx, cancel := requestContext(c)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, profile_id, site_key, survey_id, page_snapshot_id, note_text, created_at, updated_at
		FROM notes
		WHERE ($1::bigint IS NULL OR profile_id = $1)
			AND ($2::text IS NULL OR site_key = $2)
			AND ($3::bigint IS NULL OR survey_id = $3)
			AND ($4::bigint IS NULL OR page_snapshot_id = $4)
		ORDER BY id DESC
		LIMIT $5 OFFSET $6
	`, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"), optionalInt64Query(c, "survey_id"), optionalInt64Query(c, "page_snapshot_id"), limit, offset)
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer rows.Close()

	items := make([]models.Note, 0)
	for rows.Next() {
		item, err := scanNote(rows)
		if err != nil {
			handleDBError(c, err)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) getNote(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanNote(s.db.QueryRowContext(ctx, `
		SELECT id, profile_id, site_key, survey_id, page_snapshot_id, note_text, created_at, updated_at
		FROM notes
		WHERE id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
			AND ($4::bigint IS NULL OR survey_id = $4)
			AND ($5::bigint IS NULL OR page_snapshot_id = $5)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"), optionalInt64Query(c, "survey_id"), optionalInt64Query(c, "page_snapshot_id")))
	if err != nil {
		handleDBError(c, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

func (s *Server) updateNote(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req noteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}
	req.NoteText = strings.TrimSpace(req.NoteText)
	if req.NoteText == "" {
		badRequest(c, "note_text is required")
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanNote(s.db.QueryRowContext(ctx, `
		UPDATE notes
		SET note_text = $2,
			updated_at = NOW()
		WHERE id = $1
			AND ($3::bigint IS NULL OR profile_id = $3)
			AND ($4::text IS NULL OR site_key = $4)
		RETURNING id, profile_id, site_key, survey_id, page_snapshot_id, note_text, created_at, updated_at
	`, id, req.NoteText, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key")))
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, row)
}

func (s *Server) deleteNote(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		DELETE FROM knowledge_chunks
		WHERE source_type = 'note'
			AND source_id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"))
	if err != nil {
		handleDBError(c, err)
		return
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM notes
		WHERE id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"))
	if err != nil {
		handleDBError(c, err)
		return
	}
	if !writeDeleteResult(c, result) {
		return
	}
	if err := tx.Commit(); err != nil {
		handleDBError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type translateRequest struct {
	ProfileID  int64   `json:"profile_id"`
	SiteKey    string  `json:"site_key"`
	SourceText string  `json:"source_text"`
	SourceLang *string `json:"source_lang"`
	TargetLang string  `json:"target_lang"`
}

func (s *Server) translateText(c *gin.Context) {
	var req translateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}

	req.SiteKey = strings.TrimSpace(req.SiteKey)
	req.SourceText = strings.TrimSpace(req.SourceText)
	req.TargetLang = strings.TrimSpace(req.TargetLang)
	if req.TargetLang == "" {
		req.TargetLang = "zh-CN"
	}

	if req.ProfileID <= 0 || req.SiteKey == "" || req.SourceText == "" {
		badRequest(c, "profile_id, site_key and source_text are required")
		return
	}
	if containsSensitiveCredential(req.SourceText) {
		badRequest(c, "source_text appears to contain credentials or tokens and will not be sent to AI")
		return
	}
	if len([]rune(req.SourceText)) > 12000 {
		badRequest(c, "source_text is too long")
		return
	}

	ctx, cancel := aiRequestContext(c)
	defer cancel()

	cached, err := scanTranslation(s.db.QueryRowContext(ctx, `
		SELECT id, profile_id, site_key, source_text, translated_text, source_lang, target_lang, created_at
		FROM translations
		WHERE profile_id = $1
			AND site_key = $2
			AND source_text = $3
			AND target_lang = $4
		ORDER BY id DESC
		LIMIT 1
	`, req.ProfileID, req.SiteKey, req.SourceText, req.TargetLang))
	if err == nil {
		c.JSON(http.StatusOK, cached)
		return
	}
	if err != sql.ErrNoRows {
		handleDBError(c, err)
		return
	}

	translated, err := s.aiClient.Chat(ctx, []ai.ChatMessage{
		{
			Role:    "system",
			Content: "You are a survey translation assistant. Translate only the provided survey question, options, and page text. Preserve question and option meaning. Do not fabricate identity information, do not submit anything, and do not process credentials.",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Translate the following text to %s. Return only the translation.\n\n%s", req.TargetLang, req.SourceText),
		},
	}, 0.2)
	if err != nil {
		serverError(c, err)
		return
	}

	row, err := scanTranslation(s.db.QueryRowContext(ctx, `
		INSERT INTO translations (profile_id, site_key, source_text, translated_text, source_lang, target_lang)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, profile_id, site_key, source_text, translated_text, source_lang, target_lang, created_at
	`, req.ProfileID, req.SiteKey, req.SourceText, translated, ptrValue(req.SourceLang), req.TargetLang))
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, row)
}

func (s *Server) listTranslations(c *gin.Context) {
	limit, offset := limitOffset(c)
	ctx, cancel := requestContext(c)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, profile_id, site_key, source_text, translated_text, source_lang, target_lang, created_at
		FROM translations
		WHERE ($1::bigint IS NULL OR profile_id = $1)
			AND ($2::text IS NULL OR site_key = $2)
		ORDER BY id DESC
		LIMIT $3 OFFSET $4
	`, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"), limit, offset)
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer rows.Close()

	items := make([]models.Translation, 0)
	for rows.Next() {
		item, err := scanTranslation(rows)
		if err != nil {
			handleDBError(c, err)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) getTranslation(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanTranslation(s.db.QueryRowContext(ctx, `
		SELECT id, profile_id, site_key, source_text, translated_text, source_lang, target_lang, created_at
		FROM translations
		WHERE id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key")))
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, row)
}

func (s *Server) deleteTranslation(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	result, err := s.db.ExecContext(ctx, `
		DELETE FROM translations
		WHERE id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"))
	if err != nil {
		handleDBError(c, err)
		return
	}
	if !writeDeleteResult(c, result) {
		return
	}
	c.Status(http.StatusNoContent)
}

func containsSensitiveCredential(text string) bool {
	lower := strings.ToLower(text)
	sensitiveMarkers := []string{
		"password",
		"passwd",
		"pwd:",
		"refresh_token",
		"access_token",
		"id_token",
		"token:",
		"cookie:",
		"set-cookie",
		"authorization:",
		"bearer ",
		"client_secret",
		"api_key",
		"apikey",
		"api key",
		"secret:",
		"captcha",
		"verification code",
		"验证码",
		"动态码",
		"短信码",
		"otp",
		"2fa",
	}
	for _, marker := range sensitiveMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

type chatRequest struct {
	ProfileID            int64    `json:"profile_id"`
	SiteKey              string   `json:"site_key"`
	SurveyID             *int64   `json:"survey_id"`
	Scope                string   `json:"scope"`
	PageText             string   `json:"page_text"`
	QuestionText         *string  `json:"question_text"`
	OptionsText          *string  `json:"options_text"`
	ProgressText         *string  `json:"progress_text"`
	QuestionType         *string  `json:"question_type"`
	ExtractionConfidence *float64 `json:"extraction_confidence"`
	ExtractedBlocks      []struct {
		Role string `json:"role"`
		Text string `json:"text"`
	} `json:"extracted_blocks"`
	UserMessage string `json:"user_message"`
}

func (s *Server) chatWithAI(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}

	req.SiteKey = strings.TrimSpace(req.SiteKey)
	req.Scope = strings.TrimSpace(req.Scope)
	req.PageText = strings.TrimSpace(req.PageText)
	req.UserMessage = strings.TrimSpace(req.UserMessage)
	if req.ProfileID <= 0 || req.SiteKey == "" || req.UserMessage == "" {
		badRequest(c, "profile_id, site_key and user_message are required")
		return
	}
	if containsSensitiveCredential(req.PageText) || containsSensitiveCredential(req.UserMessage) {
		badRequest(c, "request appears to contain credentials or tokens and will not be sent to AI")
		return
	}
	if len([]rune(req.PageText)) > 20000 {
		req.PageText = string([]rune(req.PageText)[:20000])
	}

	ctx, cancel := aiRequestContext(c)
	defer cancel()

	notes, err := s.recentNoteContext(ctx, req.ProfileID, req.SiteKey, req.SurveyID)
	if err != nil {
		handleDBError(c, err)
		return
	}
	snapshots, err := s.recentSnapshotContext(ctx, req.ProfileID, req.SiteKey, req.SurveyID)
	if err != nil {
		handleDBError(c, err)
		return
	}
	personas, err := s.recentPersonaContext(ctx, req.ProfileID, req.SiteKey)
	if err != nil {
		handleDBError(c, err)
		return
	}
	knowledge, err := s.relatedKnowledgeContext(ctx, req.ProfileID, req.SiteKey, req.SurveyID, req.Scope, req.UserMessage)
	if err != nil {
		handleDBError(c, err)
		return
	}

	answer, err := s.aiClient.Chat(ctx, []ai.ChatMessage{
		{
			Role: "system",
			Content: `你是一个问卷调查辅助助手。

你的任务是帮助用户快速理解当前问卷页面，包括翻译、解释题目、总结页面、整理笔记、检索历史记录、对比选项，并在用户明确要求时基于用户已经提供的真实情况给出选择建议。

你只能基于系统提供的当前页面内容、当前站点人设库、当前 Profile 历史笔记、当前网站历史记录和当前问卷上下文回答。
你不能编造页面中没有的信息。
你不能编造用户身份、经历、收入、年龄、地区、消费习惯等事实。
你不能替用户自动提交问卷，也不能指导批量操作、绕过风控或处理验证码。
你不能提供绕过风控、批量作弊、自动提交问卷的方案。
你不能处理账号密码、cookie、refresh token、API key 等敏感凭证。
如果用户要求“帮我选择/推荐选项”，你可以：
1. 先解释题目和各选项含义；
2. 优先参考当前页面内容，其次参考站点人设库、用户消息和历史笔记做建议；
3. 如果人设库缺失、冲突或不足以判断，必须说明需要用户确认，不要猜测；
4. 输出“建议选择/不确定/需要你确认”，不得声称已经替用户作答。`,
		},
		{
			Role:    "user",
			Content: buildChatContext(req, personas, notes, snapshots, knowledge),
		},
	}, 0.3)
	if err != nil {
		serverError(c, err)
		return
	}

	row, err := scanAIConversation(s.db.QueryRowContext(ctx, `
		INSERT INTO ai_conversations (profile_id, site_key, survey_id, user_message, ai_message)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, profile_id, site_key, survey_id, user_message, ai_message, created_at
	`, req.ProfileID, req.SiteKey, ptrValue(req.SurveyID), req.UserMessage, answer))
	if err != nil {
		handleDBError(c, err)
		return
	}
	s.createKnowledgeFromAIConversation(ctx, row)

	c.JSON(http.StatusOK, row)
}

func (s *Server) recentNoteContext(ctx context.Context, profileID int64, siteKey string, surveyID *int64) (string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT note_text, created_at
		FROM notes
		WHERE profile_id = $1
			AND site_key = $2
			AND ($3::bigint IS NULL OR survey_id = $3)
		ORDER BY id DESC
		LIMIT 8
	`, profileID, siteKey, ptrValue(surveyID))
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var builder strings.Builder
	for rows.Next() {
		var text string
		var createdAt time.Time
		if err := rows.Scan(&text, &createdAt); err != nil {
			return "", err
		}
		builder.WriteString("- ")
		builder.WriteString(createdAt.Format(time.RFC3339))
		builder.WriteString(": ")
		builder.WriteString(trimForContext(text, 800))
		builder.WriteString("\n")
	}
	return builder.String(), rows.Err()
}

func (s *Server) recentSnapshotContext(ctx context.Context, profileID int64, siteKey string, surveyID *int64) (string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT page_title, question_text, options_text, created_at
		FROM page_snapshots
		WHERE profile_id = $1
			AND site_key = $2
			AND ($3::bigint IS NULL OR survey_id = $3)
		ORDER BY id DESC
		LIMIT 5
	`, profileID, siteKey, ptrValue(surveyID))
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var builder strings.Builder
	for rows.Next() {
		var title, question, options sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&title, &question, &options, &createdAt); err != nil {
			return "", err
		}
		builder.WriteString("- ")
		builder.WriteString(createdAt.Format(time.RFC3339))
		builder.WriteString(" | ")
		builder.WriteString(trimForContext(title.String, 200))
		if question.Valid && question.String != "" {
			builder.WriteString(" | Question: ")
			builder.WriteString(trimForContext(question.String, 500))
		}
		if options.Valid && options.String != "" {
			builder.WriteString(" | Options: ")
			builder.WriteString(trimForContext(options.String, 500))
		}
		builder.WriteString("\n")
	}
	return builder.String(), rows.Err()
}

func (s *Server) recentPersonaContext(ctx context.Context, profileID int64, siteKey string) (string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT category, persona_key, persona_value, confidence, updated_at
		FROM site_personas
		WHERE profile_id = $1
			AND site_key = $2
		ORDER BY
			CASE category
				WHEN 'basic' THEN 0
				WHEN 'demographic' THEN 1
				WHEN 'screening' THEN 2
				WHEN 'consumer' THEN 3
				WHEN 'device' THEN 4
				WHEN 'lifestyle' THEN 5
				WHEN 'preference' THEN 6
				WHEN 'avoid' THEN 7
				ELSE 8
			END ASC,
			updated_at DESC
		LIMIT 20
	`, profileID, siteKey)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var builder strings.Builder
	for rows.Next() {
		var category, key, value string
		var confidence float64
		var updatedAt time.Time
		if err := rows.Scan(&category, &key, &value, &confidence, &updatedAt); err != nil {
			return "", err
		}
		if containsSensitiveCredential(value) {
			continue
		}
		builder.WriteString("- [")
		builder.WriteString(category)
		builder.WriteString("] ")
		builder.WriteString(key)
		builder.WriteString(": ")
		builder.WriteString(trimForContext(value, 500))
		builder.WriteString(" (confidence ")
		builder.WriteString(strconv.FormatFloat(confidence, 'f', 2, 64))
		builder.WriteString(", updated ")
		builder.WriteString(updatedAt.Format(time.RFC3339))
		builder.WriteString(")\n")
	}
	return builder.String(), rows.Err()
}

func buildChatContext(req chatRequest, personas, notes, snapshots, knowledge string) string {
	var builder strings.Builder
	builder.WriteString("当前 Profile ID：")
	builder.WriteString(strconv.FormatInt(req.ProfileID, 10))
	builder.WriteString("\n当前网站：")
	builder.WriteString(req.SiteKey)
	builder.WriteString("\n当前问卷：")
	if req.SurveyID != nil {
		builder.WriteString(strconv.FormatInt(*req.SurveyID, 10))
	} else {
		builder.WriteString("未识别")
	}
	builder.WriteString("\n知识库范围：")
	if req.Scope == "" {
		builder.WriteString("site")
	} else {
		builder.WriteString(req.Scope)
	}
	builder.WriteString("\n\n当前题目：\n")
	builder.WriteString(trimForContext(ptrValueString(req.QuestionText), 2000))
	builder.WriteString("\n\n当前选项：\n")
	builder.WriteString(trimForContext(ptrValueString(req.OptionsText), 2000))
	builder.WriteString("\n\n当前进度：\n")
	builder.WriteString(emptyFallback(ptrValueString(req.ProgressText)))
	builder.WriteString("\n\n题目类型：\n")
	builder.WriteString(emptyFallback(ptrValueString(req.QuestionType)))
	if req.ExtractionConfidence != nil {
		builder.WriteString("\n\n页面识别置信度：\n")
		builder.WriteString(strconv.FormatFloat(*req.ExtractionConfidence, 'f', 2, 64))
	}
	if len(req.ExtractedBlocks) > 0 {
		builder.WriteString("\n\n结构化页面块：\n")
		for _, block := range req.ExtractedBlocks {
			role := strings.TrimSpace(block.Role)
			text := strings.TrimSpace(block.Text)
			if role == "" || text == "" || containsSensitiveCredential(text) {
				continue
			}
			builder.WriteString("- ")
			builder.WriteString(role)
			builder.WriteString(": ")
			builder.WriteString(trimForContext(text, 500))
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\n\n当前页面补充文本：\n")
	builder.WriteString(trimForContext(req.PageText, 12000))
	builder.WriteString("\n\n当前站点人设库：\n")
	builder.WriteString(emptyFallback(personas))
	builder.WriteString("\n\n相关历史笔记：\n")
	builder.WriteString(emptyFallback(notes))
	builder.WriteString("\n\n相关历史页面：\n")
	builder.WriteString(emptyFallback(snapshots))
	builder.WriteString("\n\n相关知识库：\n")
	builder.WriteString(emptyFallback(knowledge))
	builder.WriteString("\n\n用户问题：\n")
	builder.WriteString(req.UserMessage)
	return builder.String()
}

func ptrValueString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nullIfEmpty(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func emptyFallback(value string) string {
	if strings.TrimSpace(value) == "" {
		return "无"
	}
	return value
}

func trimForContext(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "\n[内容已截断]"
}

func vectorLiteral(values []float64) (string, bool) {
	if len(values) != knowledgeEmbeddingDimensions {
		return "", false
	}
	var builder strings.Builder
	builder.WriteString("[")
	for index, value := range values {
		if index > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
	}
	builder.WriteString("]")
	return builder.String(), true
}

func (s *Server) knowledgeEmbeddingLiteral(ctx context.Context, text string) *string {
	text = trimForContext(text, 6000)
	if text == "" || !s.aiClient.EmbeddingsConfigured() {
		return nil
	}
	embedding, err := s.aiClient.Embedding(ctx, text)
	if err != nil {
		return nil
	}
	literal, ok := vectorLiteral(embedding)
	if !ok {
		return nil
	}
	return &literal
}

func (s *Server) listAIConversations(c *gin.Context) {
	limit, offset := limitOffset(c)
	ctx, cancel := requestContext(c)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, profile_id, site_key, survey_id, user_message, ai_message, created_at
		FROM ai_conversations
		WHERE ($1::bigint IS NULL OR profile_id = $1)
			AND ($2::text IS NULL OR site_key = $2)
			AND ($3::bigint IS NULL OR survey_id = $3)
		ORDER BY id DESC
		LIMIT $4 OFFSET $5
	`, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"), optionalInt64Query(c, "survey_id"), limit, offset)
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer rows.Close()

	items := make([]models.AIConversation, 0)
	for rows.Next() {
		item, err := scanAIConversation(rows)
		if err != nil {
			handleDBError(c, err)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) getAIConversation(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanAIConversation(s.db.QueryRowContext(ctx, `
		SELECT id, profile_id, site_key, survey_id, user_message, ai_message, created_at
		FROM ai_conversations
		WHERE id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
			AND ($4::bigint IS NULL OR survey_id = $4)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"), optionalInt64Query(c, "survey_id")))
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, row)
}

func (s *Server) deleteAIConversation(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		DELETE FROM knowledge_chunks
		WHERE source_type = 'ai_conversation'
			AND source_id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"))
	if err != nil {
		handleDBError(c, err)
		return
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM ai_conversations
		WHERE id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
			AND ($4::bigint IS NULL OR survey_id = $4)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"), optionalInt64Query(c, "survey_id"))
	if err != nil {
		handleDBError(c, err)
		return
	}
	if !writeDeleteResult(c, result) {
		return
	}
	if err := tx.Commit(); err != nil {
		handleDBError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type personaRequest struct {
	ProfileID    int64   `json:"profile_id"`
	SiteKey      string  `json:"site_key"`
	Category     string  `json:"category"`
	PersonaKey   string  `json:"persona_key"`
	PersonaValue string  `json:"persona_value"`
	Confidence   float64 `json:"confidence"`
	SourceType   string  `json:"source_type"`
	SourceID     *int64  `json:"source_id"`
}

func (s *Server) createSitePersona(c *gin.Context) {
	var req personaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}
	if !preparePersonaRequest(c, &req, true) {
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanSitePersona(s.db.QueryRowContext(ctx, `
		INSERT INTO site_personas (profile_id, site_key, category, persona_key, persona_value, confidence, source_type, source_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (profile_id, site_key, persona_key) DO UPDATE SET
			category = EXCLUDED.category,
			persona_value = EXCLUDED.persona_value,
			confidence = EXCLUDED.confidence,
			source_type = EXCLUDED.source_type,
			source_id = EXCLUDED.source_id,
			updated_at = NOW()
		RETURNING id, profile_id, site_key, category, persona_key, persona_value, confidence, source_type, source_id, created_at, updated_at
	`, req.ProfileID, req.SiteKey, req.Category, req.PersonaKey, req.PersonaValue, req.Confidence, req.SourceType, ptrValue(req.SourceID)))
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusCreated, row)
}

func (s *Server) listSitePersonas(c *gin.Context) {
	limit, offset := limitOffset(c)
	ctx, cancel := requestContext(c)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, profile_id, site_key, category, persona_key, persona_value, confidence, source_type, source_id, created_at, updated_at
		FROM site_personas
		WHERE ($1::bigint IS NULL OR profile_id = $1)
			AND ($2::text IS NULL OR site_key = $2)
			AND ($3::text IS NULL OR category = $3)
		ORDER BY updated_at DESC, id DESC
		LIMIT $4 OFFSET $5
	`, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"), optionalStringQuery(c, "category"), limit, offset)
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer rows.Close()

	items, err := readSitePersonaRows(rows)
	if err != nil {
		handleDBError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) sitePersonaContext(c *gin.Context) {
	profileID, ok := requiredProfileID(c)
	if !ok {
		return
	}
	siteKey := strings.TrimSpace(c.Query("site_key"))
	if siteKey == "" {
		badRequest(c, "site_key is required")
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	contextText, err := s.recentPersonaContext(ctx, profileID, siteKey)
	if err != nil {
		handleDBError(c, err)
		return
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, profile_id, site_key, category, persona_key, persona_value, confidence, source_type, source_id, created_at, updated_at
		FROM site_personas
		WHERE profile_id = $1
			AND site_key = $2
			AND ($3::text IS NULL OR category = $3)
		ORDER BY updated_at DESC, id DESC
		LIMIT 100
	`, profileID, siteKey, optionalStringQuery(c, "category"))
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer rows.Close()

	items, err := readSitePersonaRows(rows)
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"context": emptyFallback(contextText),
		"items":   items,
	})
}

func (s *Server) updateSitePersona(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req personaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}
	if !preparePersonaRequest(c, &req, false) {
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	row, err := scanSitePersona(s.db.QueryRowContext(ctx, `
		UPDATE site_personas
		SET category = $2,
			persona_key = $3,
			persona_value = $4,
			confidence = $5,
			source_type = $6,
			source_id = $7,
			updated_at = NOW()
		WHERE id = $1
			AND ($8::bigint IS NULL OR profile_id = $8)
			AND ($9::text IS NULL OR site_key = $9)
		RETURNING id, profile_id, site_key, category, persona_key, persona_value, confidence, source_type, source_id, created_at, updated_at
	`, id, req.Category, req.PersonaKey, req.PersonaValue, req.Confidence, req.SourceType, ptrValue(req.SourceID), optionalInt64Body(req.ProfileID), optionalStringBody(req.SiteKey)))
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, row)
}

func (s *Server) deleteSitePersona(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	result, err := s.db.ExecContext(ctx, `
		DELETE FROM site_personas
		WHERE id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"))
	if err != nil {
		handleDBError(c, err)
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		handleDBError(c, err)
		return
	}
	if rowsAffected == 0 {
		notFound(c)
		return
	}
	c.Status(http.StatusNoContent)
}

func readSitePersonaRows(rows *sql.Rows) ([]models.SitePersona, error) {
	items := make([]models.SitePersona, 0)
	for rows.Next() {
		item, err := scanSitePersona(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func preparePersonaRequest(c *gin.Context, req *personaRequest, requireProfileAndSite bool) bool {
	req.SiteKey = strings.TrimSpace(req.SiteKey)
	req.Category = normalizePersonaCategory(req.Category)
	req.PersonaValue = strings.TrimSpace(req.PersonaValue)
	req.PersonaKey = normalizePersonaKey(req.PersonaKey)
	if req.PersonaKey == "" {
		req.PersonaKey = inferPersonaKey(req.PersonaValue)
	}
	req.SourceType = strings.TrimSpace(req.SourceType)
	if req.SourceType == "" {
		req.SourceType = "manual"
	}
	if req.Confidence <= 0 {
		req.Confidence = 1
	}
	if req.Confidence > 1 {
		req.Confidence = 1
	}
	if requireProfileAndSite && (req.ProfileID <= 0 || req.SiteKey == "") {
		badRequest(c, "profile_id and site_key are required")
		return false
	}
	if req.PersonaValue == "" {
		badRequest(c, "persona_value is required")
		return false
	}
	if req.PersonaKey == "" {
		badRequest(c, "persona_key is required")
		return false
	}
	if containsSensitiveCredential(req.PersonaKey) || containsSensitiveCredential(req.PersonaValue) {
		badRequest(c, "persona appears to contain credentials, tokens or verification codes")
		return false
	}
	return true
}

func normalizePersonaCategory(category string) string {
	category = strings.TrimSpace(category)
	switch category {
	case "basic", "demographic", "consumer", "device", "lifestyle", "screening", "preference", "avoid":
		return category
	default:
		return "preference"
	}
}

func normalizePersonaKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "：", ":")
	if strings.Contains(key, ":") {
		key = strings.SplitN(key, ":", 2)[0]
	}
	key = regexp.MustCompile(`[^a-z0-9_\-]+`).ReplaceAllString(key, "_")
	key = strings.Trim(key, "_-")
	if len(key) > 120 {
		key = key[:120]
	}
	return key
}

func inferPersonaKey(value string) string {
	value = strings.TrimSpace(value)
	if strings.Contains(value, ":") || strings.Contains(value, "：") {
		return normalizePersonaKey(value)
	}
	lower := strings.ToLower(value)
	switch {
	case strings.Contains(value, "饮料") || strings.Contains(value, "咖啡") || strings.Contains(value, "茶") || strings.Contains(lower, "drink"):
		return "drink_preference"
	case strings.Contains(value, "手机") || strings.Contains(value, "电脑") || strings.Contains(lower, "iphone") || strings.Contains(lower, "android") || strings.Contains(lower, "windows") || strings.Contains(lower, "mac"):
		return "device"
	case strings.Contains(value, "购物") || strings.Contains(value, "网购") || strings.Contains(lower, "shopping"):
		return "shopping_preference"
	case strings.Contains(value, "收入") || strings.Contains(value, "年龄") || strings.Contains(value, "城市") || strings.Contains(value, "学历"):
		return "demographic_profile"
	default:
		hash := 0
		for _, r := range []rune(value) {
			hash = (hash*31 + int(r)) % 1000000
		}
		return fmt.Sprintf("manual_%06d", hash)
	}
}

func optionalInt64Body(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

func optionalStringBody(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

type knowledgeRequest struct {
	ProfileID  int64  `json:"profile_id"`
	SiteKey    string `json:"site_key"`
	SurveyID   *int64 `json:"survey_id"`
	Scope      string `json:"scope"`
	SourceType string `json:"source_type"`
	SourceID   *int64 `json:"source_id"`
	ChunkText  string `json:"chunk_text"`
}

func (s *Server) createKnowledgeChunk(c *gin.Context) {
	var req knowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid JSON body")
		return
	}
	req.SiteKey = strings.TrimSpace(req.SiteKey)
	req.Scope = normalizeScope(req.Scope, req.SurveyID)
	req.SourceType = strings.TrimSpace(req.SourceType)
	if req.SourceType == "" {
		req.SourceType = "manual"
	}
	req.ChunkText = strings.TrimSpace(req.ChunkText)
	if req.ProfileID <= 0 || req.SiteKey == "" || req.ChunkText == "" {
		badRequest(c, "profile_id, site_key and chunk_text are required")
		return
	}
	if containsSensitiveCredential(req.ChunkText) {
		badRequest(c, "chunk_text appears to contain credentials or tokens")
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	embedding := s.knowledgeEmbeddingLiteral(ctx, req.ChunkText)
	row, err := scanKnowledgeChunk(s.db.QueryRowContext(ctx, `
		INSERT INTO knowledge_chunks (profile_id, site_key, survey_id, scope, source_type, source_id, chunk_text, embedding)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::vector)
		RETURNING id, profile_id, site_key, survey_id, scope, source_type, source_id, chunk_text, created_at
	`, req.ProfileID, req.SiteKey, ptrValue(req.SurveyID), req.Scope, req.SourceType, ptrValue(req.SourceID), req.ChunkText, ptrValue(embedding)))
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusCreated, row)
}

func (s *Server) listKnowledgeChunks(c *gin.Context) {
	limit, offset := limitOffset(c)
	ctx, cancel := requestContext(c)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, profile_id, site_key, survey_id, scope, source_type, source_id, chunk_text, created_at
		FROM knowledge_chunks
		WHERE ($1::bigint IS NULL OR profile_id = $1)
			AND ($2::text IS NULL OR site_key = $2)
			AND ($3::bigint IS NULL OR survey_id = $3)
			AND ($4::text IS NULL OR scope = $4)
		ORDER BY id DESC
		LIMIT $5 OFFSET $6
	`, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"), optionalInt64Query(c, "survey_id"), optionalStringQuery(c, "scope"), limit, offset)
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer rows.Close()

	items, err := readKnowledgeRows(rows)
	if err != nil {
		handleDBError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) searchKnowledgeChunks(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		badRequest(c, "q is required")
		return
	}
	limit, offset := limitOffset(c)
	ctx, cancel := requestContext(c)
	defer cancel()

	embedding := s.knowledgeEmbeddingLiteral(ctx, query)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, profile_id, site_key, survey_id, scope, source_type, source_id, chunk_text, created_at
		FROM knowledge_chunks
		WHERE ($1::bigint IS NULL OR profile_id = $1)
			AND ($2::text IS NULL OR site_key = $2)
			AND ($3::bigint IS NULL OR survey_id = $3)
			AND ($4::text IS NULL OR scope = $4)
			AND (
				($8::vector IS NOT NULL AND embedding IS NOT NULL)
				OR search_vector @@ plainto_tsquery('simple', $5)
				OR chunk_text ILIKE '%' || $5 || '%'
			)
		ORDER BY
			CASE source_type
				WHEN 'note' THEN 0
				WHEN 'page_snapshot' THEN 1
				WHEN 'ai_conversation' THEN 2
				ELSE 3
			END ASC,
			CASE
				WHEN $3::bigint IS NOT NULL AND survey_id = $3 THEN 0
				WHEN scope = 'site' THEN 1
				WHEN scope = 'profile' THEN 2
				ELSE 3
			END ASC,
			CASE
				WHEN $8::vector IS NOT NULL AND embedding IS NOT NULL THEN embedding <=> $8::vector
				ELSE NULL
			END ASC NULLS LAST,
			ts_rank_cd(search_vector, plainto_tsquery('simple', $5)) DESC,
			id DESC
		LIMIT $6 OFFSET $7
	`, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"), optionalInt64Query(c, "survey_id"), optionalStringQuery(c, "scope"), query, limit, offset, ptrValue(embedding))
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer rows.Close()

	items, err := readKnowledgeRows(rows)
	if err != nil {
		handleDBError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) deleteKnowledgeChunk(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	result, err := s.db.ExecContext(ctx, `
		DELETE FROM knowledge_chunks
		WHERE id = $1
			AND ($2::bigint IS NULL OR profile_id = $2)
			AND ($3::text IS NULL OR site_key = $3)
	`, id, optionalInt64Query(c, "profile_id"), optionalStringQuery(c, "site_key"))
	if err != nil {
		handleDBError(c, err)
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		handleDBError(c, err)
		return
	}
	if rowsAffected == 0 {
		notFound(c)
		return
	}
	c.Status(http.StatusNoContent)
}

func readKnowledgeRows(rows *sql.Rows) ([]models.KnowledgeChunk, error) {
	items := make([]models.KnowledgeChunk, 0)
	for rows.Next() {
		item, err := scanKnowledgeChunk(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) createKnowledgeFromPageSnapshot(ctx context.Context, snapshot models.PageSnapshot) {
	textParts := []string{}
	if snapshot.PageTitle != nil && *snapshot.PageTitle != "" {
		textParts = append(textParts, "Title: "+*snapshot.PageTitle)
	}
	if snapshot.QuestionText != nil && *snapshot.QuestionText != "" {
		textParts = append(textParts, "Question: "+*snapshot.QuestionText)
	}
	if snapshot.OptionsText != nil && *snapshot.OptionsText != "" {
		textParts = append(textParts, "Options: "+*snapshot.OptionsText)
	}
	text := strings.TrimSpace(strings.Join(textParts, "\n"))
	if text == "" || containsSensitiveCredential(text) {
		return
	}
	embedding := s.knowledgeEmbeddingLiteral(ctx, text)
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO knowledge_chunks (profile_id, site_key, survey_id, scope, source_type, source_id, chunk_text, embedding)
		VALUES ($1, $2, $3, $4, 'page_snapshot', $5, $6, $7::vector)
	`, snapshot.ProfileID, snapshot.SiteKey, ptrValue(snapshot.SurveyID), normalizeScope("", snapshot.SurveyID), snapshot.ID, text, ptrValue(embedding))
}

func (s *Server) createKnowledgeFromNote(ctx context.Context, note models.Note) {
	if strings.TrimSpace(note.NoteText) == "" || containsSensitiveCredential(note.NoteText) {
		return
	}
	embedding := s.knowledgeEmbeddingLiteral(ctx, note.NoteText)
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO knowledge_chunks (profile_id, site_key, survey_id, scope, source_type, source_id, chunk_text, embedding)
		VALUES ($1, $2, $3, $4, 'note', $5, $6, $7::vector)
	`, note.ProfileID, note.SiteKey, ptrValue(note.SurveyID), normalizeScope("", note.SurveyID), note.ID, note.NoteText, ptrValue(embedding))
}

func (s *Server) createKnowledgeFromAIConversation(ctx context.Context, conversation models.AIConversation) {
	if !shouldStoreAIConversationKnowledge(conversation.UserMessage, conversation.AIMessage) {
		return
	}
	text := strings.TrimSpace("User: " + conversation.UserMessage + "\nAI: " + conversation.AIMessage)
	if text == "" || containsSensitiveCredential(text) {
		return
	}
	text = trimForContext(text, 3000)
	embedding := s.knowledgeEmbeddingLiteral(ctx, text)
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO knowledge_chunks (profile_id, site_key, survey_id, scope, source_type, source_id, chunk_text, embedding)
		VALUES ($1, $2, $3, $4, 'ai_conversation', $5, $6, $7::vector)
	`, conversation.ProfileID, conversation.SiteKey, ptrValue(conversation.SurveyID), normalizeScope("", conversation.SurveyID), conversation.ID, text, ptrValue(embedding))
}

func shouldStoreAIConversationKnowledge(userMessage, aiMessage string) bool {
	combined := strings.TrimSpace(userMessage + "\n" + aiMessage)
	if len([]rune(combined)) < 80 {
		return false
	}
	lower := strings.ToLower(combined)
	skipMarkers := []string{"hello", "hi", "你好", "在吗", "test", "ok"}
	for _, marker := range skipMarkers {
		if strings.TrimSpace(lower) == marker {
			return false
		}
	}
	keepMarkers := []string{"question", "option", "survey", "answer", "translate", "题", "选项", "问卷", "翻译", "解释", "建议", "选择"}
	for _, marker := range keepMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return len([]rune(combined)) >= 220
}

func (s *Server) relatedKnowledgeContext(ctx context.Context, profileID int64, siteKey string, surveyID *int64, scope, query string) (string, error) {
	scope = normalizeScope(scope, surveyID)
	query = strings.TrimSpace(query)
	if query == "" {
		return "", nil
	}

	embedding := s.knowledgeEmbeddingLiteral(ctx, query)
	rows, err := s.db.QueryContext(ctx, `
		SELECT chunk_text, source_type, created_at
		FROM knowledge_chunks
		WHERE profile_id = $1
			AND site_key = $2
			AND ($3::bigint IS NULL OR survey_id = $3)
			AND ($4::text IS NULL OR scope = $4)
			AND (
				($6::vector IS NOT NULL AND embedding IS NOT NULL)
				OR search_vector @@ plainto_tsquery('simple', $5)
				OR chunk_text ILIKE '%' || $5 || '%'
			)
		ORDER BY
			CASE source_type
				WHEN 'note' THEN 0
				WHEN 'page_snapshot' THEN 1
				WHEN 'ai_conversation' THEN 2
				ELSE 3
			END ASC,
			CASE
				WHEN $6::vector IS NOT NULL AND embedding IS NOT NULL THEN embedding <=> $6::vector
				ELSE NULL
			END ASC NULLS LAST,
			ts_rank_cd(search_vector, plainto_tsquery('simple', $5)) DESC,
			id DESC
		LIMIT 5
	`, profileID, siteKey, ptrValue(surveyID), scope, query, ptrValue(embedding))
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var builder strings.Builder
	for rows.Next() {
		var text, sourceType string
		var createdAt time.Time
		if err := rows.Scan(&text, &sourceType, &createdAt); err != nil {
			return "", err
		}
		builder.WriteString("- ")
		builder.WriteString(createdAt.Format(time.RFC3339))
		builder.WriteString(" | ")
		builder.WriteString(sourceType)
		builder.WriteString(": ")
		builder.WriteString(trimForContext(text, 900))
		builder.WriteString("\n")
	}
	return builder.String(), rows.Err()
}

func normalizeScope(scope string, surveyID *int64) string {
	scope = strings.TrimSpace(scope)
	switch scope {
	case "survey", "site", "profile", "team":
		return scope
	}
	if surveyID != nil {
		return "survey"
	}
	return "site"
}

func (s *Server) exportMarkdown(c *gin.Context) {
	profileID, ok := requiredProfileID(c)
	if !ok {
		return
	}
	siteKey := strings.TrimSpace(c.Query("site_key"))

	ctx, cancel := requestContext(c)
	defer cancel()

	var builder strings.Builder
	builder.WriteString("# Survey AI Workspace Export\n\n")
	builder.WriteString("- Profile ID: ")
	builder.WriteString(strconv.FormatInt(profileID, 10))
	builder.WriteString("\n")
	if siteKey != "" {
		builder.WriteString("- Site: ")
		builder.WriteString(siteKey)
		builder.WriteString("\n")
	}
	builder.WriteString("- Exported At: ")
	builder.WriteString(time.Now().Format(time.RFC3339))
	builder.WriteString("\n\n")

	if err := s.writeMarkdownSection(ctx, &builder, "Surveys", `
		SELECT id, site_key, COALESCE(survey_title, ''), COALESCE(status, ''), created_at
		FROM surveys
		WHERE profile_id = $1 AND ($2::text = '' OR site_key = $2)
		ORDER BY id DESC
		LIMIT 100
	`, profileID, siteKey); err != nil {
		handleDBError(c, err)
		return
	}
	if err := s.writeMarkdownSection(ctx, &builder, "Notes", `
		SELECT id, site_key, COALESCE(note_text, ''), '', created_at
		FROM notes
		WHERE profile_id = $1 AND ($2::text = '' OR site_key = $2)
		ORDER BY id DESC
		LIMIT 100
	`, profileID, siteKey); err != nil {
		handleDBError(c, err)
		return
	}
	if err := s.writeMarkdownSection(ctx, &builder, "AI Conversations", `
		SELECT id, site_key, COALESCE(user_message, ''), COALESCE(ai_message, ''), created_at
		FROM ai_conversations
		WHERE profile_id = $1 AND ($2::text = '' OR site_key = $2)
		ORDER BY id DESC
		LIMIT 100
	`, profileID, siteKey); err != nil {
		handleDBError(c, err)
		return
	}
	if err := s.writeMarkdownSection(ctx, &builder, "Knowledge", `
		SELECT id, site_key, COALESCE(source_type, ''), COALESCE(chunk_text, ''), created_at
		FROM knowledge_chunks
		WHERE profile_id = $1 AND ($2::text = '' OR site_key = $2)
		ORDER BY id DESC
		LIMIT 100
	`, profileID, siteKey); err != nil {
		handleDBError(c, err)
		return
	}

	filename := "survey-ai-export-profile-" + strconv.FormatInt(profileID, 10) + ".md"
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(builder.String()))
}

func (s *Server) exportNotesCSV(c *gin.Context) {
	profileID, ok := requiredProfileID(c)
	if !ok {
		return
	}
	siteKey := strings.TrimSpace(c.Query("site_key"))

	ctx, cancel := requestContext(c)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, profile_id, site_key, COALESCE(survey_id, 0), COALESCE(page_snapshot_id, 0), note_text, created_at
		FROM notes
		WHERE profile_id = $1 AND ($2::text = '' OR site_key = $2)
		ORDER BY id DESC
	`, profileID, siteKey)
	if err != nil {
		handleDBError(c, err)
		return
	}
	defer rows.Close()

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write([]string{"id", "profile_id", "site_key", "survey_id", "page_snapshot_id", "note_text", "created_at"})
	for rows.Next() {
		var id, rowProfileID, surveyID, pageSnapshotID int64
		var rowSiteKey, noteText string
		var createdAt time.Time
		if err := rows.Scan(&id, &rowProfileID, &rowSiteKey, &surveyID, &pageSnapshotID, &noteText, &createdAt); err != nil {
			handleDBError(c, err)
			return
		}
		_ = writer.Write([]string{
			strconv.FormatInt(id, 10),
			strconv.FormatInt(rowProfileID, 10),
			rowSiteKey,
			strconv.FormatInt(surveyID, 10),
			strconv.FormatInt(pageSnapshotID, 10),
			noteText,
			createdAt.Format(time.RFC3339),
		})
	}
	if err := rows.Err(); err != nil {
		handleDBError(c, err)
		return
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		serverError(c, err)
		return
	}

	filename := "survey-ai-notes-profile-" + strconv.FormatInt(profileID, 10) + ".csv"
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buffer.Bytes())
}

func (s *Server) exportDailyReport(c *gin.Context) {
	profileID := optionalInt64Query(c, "profile_id")
	siteKey := strings.TrimSpace(c.Query("site_key"))

	ctx, cancel := requestContext(c)
	defer cancel()

	var builder strings.Builder
	builder.WriteString("# Survey AI Daily Report\n\n")
	builder.WriteString("- Date: ")
	builder.WriteString(time.Now().Format("2006-01-02"))
	builder.WriteString("\n")
	if profileID != nil {
		builder.WriteString("- Profile ID: ")
		builder.WriteString(strconv.FormatInt(profileID.(int64), 10))
		builder.WriteString("\n")
	}
	if siteKey != "" {
		builder.WriteString("- Site: ")
		builder.WriteString(siteKey)
		builder.WriteString("\n")
	}
	builder.WriteString("\n")

	if err := s.writeDailyCount(ctx, &builder, "New Page Snapshots", "page_snapshots", profileID, siteKey); err != nil {
		handleDBError(c, err)
		return
	}
	if err := s.writeDailyCount(ctx, &builder, "New Notes", "notes", profileID, siteKey); err != nil {
		handleDBError(c, err)
		return
	}
	if err := s.writeDailyCount(ctx, &builder, "New Translations", "translations", profileID, siteKey); err != nil {
		handleDBError(c, err)
		return
	}
	if err := s.writeDailyCount(ctx, &builder, "New AI Conversations", "ai_conversations", profileID, siteKey); err != nil {
		handleDBError(c, err)
		return
	}

	c.Header("Content-Disposition", `attachment; filename="survey-ai-daily-report.md"`)
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(builder.String()))
}

func requiredProfileID(c *gin.Context) (int64, bool) {
	value := strings.TrimSpace(c.Query("profile_id"))
	if value == "" {
		badRequest(c, "profile_id is required")
		return 0, false
	}
	profileID, err := strconv.ParseInt(value, 10, 64)
	if err != nil || profileID <= 0 {
		badRequest(c, "profile_id is invalid")
		return 0, false
	}
	return profileID, true
}

func (s *Server) writeMarkdownSection(ctx context.Context, builder *strings.Builder, title, query string, profileID int64, siteKey string) error {
	builder.WriteString("## ")
	builder.WriteString(title)
	builder.WriteString("\n\n")

	rows, err := s.db.QueryContext(ctx, query, profileID, siteKey)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int64
		var rowSiteKey, first, second string
		var createdAt time.Time
		if err := rows.Scan(&id, &rowSiteKey, &first, &second, &createdAt); err != nil {
			return err
		}
		count++
		builder.WriteString("### #")
		builder.WriteString(strconv.FormatInt(id, 10))
		builder.WriteString(" ")
		builder.WriteString(rowSiteKey)
		builder.WriteString("\n\n")
		builder.WriteString("- Created: ")
		builder.WriteString(createdAt.Format(time.RFC3339))
		builder.WriteString("\n\n")
		if first != "" {
			builder.WriteString(trimForContext(first, 2000))
			builder.WriteString("\n\n")
		}
		if second != "" {
			builder.WriteString(trimForContext(second, 2000))
			builder.WriteString("\n\n")
		}
	}
	if count == 0 {
		builder.WriteString("No records.\n\n")
	}
	return rows.Err()
}

func (s *Server) writeDailyCount(ctx context.Context, builder *strings.Builder, label, table string, profileID any, siteKey string) error {
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s
		WHERE created_at >= CURRENT_DATE
			AND ($1::bigint IS NULL OR profile_id = $1)
			AND ($2::text = '' OR site_key = $2)
	`, table)
	var count int64
	if err := s.db.QueryRowContext(ctx, query, profileID, siteKey).Scan(&count); err != nil {
		return err
	}
	builder.WriteString("- ")
	builder.WriteString(label)
	builder.WriteString(": ")
	builder.WriteString(strconv.FormatInt(count, 10))
	builder.WriteString("\n")
	return nil
}
