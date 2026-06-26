package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const authTokenTTL = 7 * 24 * time.Hour

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authTokenPayload struct {
	Username string `json:"username"`
	Expires  int64  `json:"expires"`
}

func (s *Server) authEnabled() bool {
	return strings.TrimSpace(s.auth.Username) != "" && strings.TrimSpace(s.auth.Password) != ""
}

func (s *Server) login(c *gin.Context) {
	if !s.authEnabled() {
		c.JSON(http.StatusOK, gin.H{
			"token":      "",
			"expires_at": time.Now().Add(authTokenTTL).Format(time.RFC3339),
			"disabled":   true,
		})
		return
	}

	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid login payload")
		return
	}

	if subtle.ConstantTimeCompare([]byte(req.Username), []byte(s.auth.Username)) != 1 ||
		subtle.ConstantTimeCompare([]byte(req.Password), []byte(s.auth.Password)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	expires := time.Now().Add(authTokenTTL)
	token, err := s.signAuthToken(authTokenPayload{Username: s.auth.Username, Expires: expires.Unix()})
	if err != nil {
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expires.Format(time.RFC3339),
	})
}

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.authEnabled() {
			c.Next()
			return
		}

		header := strings.TrimSpace(c.GetHeader("Authorization"))
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if token == "" || !s.verifyAuthToken(token) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "login required"})
			return
		}

		c.Next()
	}
}

func (s *Server) signAuthToken(payload authTokenPayload) (string, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signature := s.authSignature(encodedPayload)
	return encodedPayload + "." + signature, nil
}

func (s *Server) verifyAuthToken(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}

	expected := s.authSignature(parts[0])
	if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expected)) != 1 {
		return false
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	var payload authTokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return false
	}
	if payload.Username != s.auth.Username || payload.Expires <= time.Now().Unix() {
		return false
	}
	return true
}

func (s *Server) authSignature(encodedPayload string) string {
	key := strings.TrimSpace(s.auth.TokenSecret)
	if key == "" {
		key = s.auth.Username + ":" + s.auth.Password
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(encodedPayload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) authStatus() gin.H {
	return gin.H{
		"auth_required": s.authEnabled(),
	}
}
