package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/helixreader/helix/server/internal/kosync"
)

type kosyncUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// kosyncError emits the Helix error envelope plus the top-level "message"
// field that KOReader's kosync plugin shows to the user (main.lua reads
// body.message on register/authorize failures).
func kosyncError(c *gin.Context, status int, public string) {
	c.AbortWithStatusJSON(status, gin.H{"error": public, "message": public, "request_id": c.GetString("request_id")})
}

func (s *Server) kosyncAvailable(c *gin.Context) bool {
	if s.KOSync == nil {
		kosyncError(c, http.StatusServiceUnavailable, "KOReader sync is not configured")
		return false
	}
	return true
}

func (s *Server) kosyncAuthenticate(c *gin.Context) (string, bool) {
	username := c.GetHeader("x-auth-user")
	if err := s.KOSync.Authenticate(c.Request.Context(), username, c.GetHeader("x-auth-key")); err != nil {
		if errors.Is(err, kosync.ErrInvalidAuth) {
			kosyncError(c, http.StatusUnauthorized, "invalid username or password")
		} else {
			kosyncError(c, http.StatusInternalServerError, "authentication failed")
		}
		return "", false
	}
	return username, true
}

func (s *Server) kosyncRegister(c *gin.Context) {
	if !s.kosyncAvailable(c) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	var request kosyncUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		kosyncError(c, http.StatusBadRequest, "invalid user request")
		return
	}
	if err := s.KOSync.Register(c.Request.Context(), request.Username, request.Password); err != nil {
		switch {
		case errors.Is(err, kosync.ErrRegistrationDisabled):
			// Reference: errors.lua 2005 => 402 (api.json register expects [201, 402]).
			kosyncError(c, http.StatusPaymentRequired, "registration is disabled")
		case errors.Is(err, kosync.ErrInvalidInput):
			kosyncError(c, http.StatusBadRequest, "invalid username or password")
		case errors.Is(err, kosync.ErrUserExists):
			// Reference: errors.lua 2002 => 402; 409 is not in api.json expected_status.
			kosyncError(c, http.StatusPaymentRequired, "Username is already registered.")
		default:
			kosyncError(c, http.StatusInternalServerError, "user creation failed")
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"username": request.Username})
}

func (s *Server) kosyncAuth(c *gin.Context) {
	if !s.kosyncAvailable(c) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024)
	if _, ok := s.kosyncAuthenticate(c); !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"authorized": "OK"})
}

func (s *Server) kosyncPutProgress(c *gin.Context) {
	if !s.kosyncAvailable(c) {
		return
	}
	username, ok := s.kosyncAuthenticate(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16384)
	var progress kosync.Progress
	if err := c.ShouldBindJSON(&progress); err != nil {
		kosyncError(c, http.StatusBadRequest, "invalid progress request")
		return
	}
	progress, err := s.KOSync.PutProgress(c.Request.Context(), username, progress)
	if errors.Is(err, kosync.ErrInvalidInput) {
		kosyncError(c, http.StatusBadRequest, "missing or invalid progress fields")
		return
	}
	if err != nil {
		kosyncError(c, http.StatusInternalServerError, "progress update failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"document": progress.Document, "timestamp": progress.Timestamp})
}

func (s *Server) kosyncGetProgress(c *gin.Context) {
	if !s.kosyncAvailable(c) {
		return
	}
	username, ok := s.kosyncAuthenticate(c)
	if !ok {
		return
	}
	progress, err := s.KOSync.GetProgress(c.Request.Context(), username, c.Param("document"))
	switch {
	case errors.Is(err, kosync.ErrProgressNotFound):
		// Reference: get_progress answers 200 with an empty object when nothing is
		// stored; api.json expects [200, 401] so a 404 is a client-side error.
		c.JSON(http.StatusOK, gin.H{})
	case errors.Is(err, kosync.ErrInvalidInput):
		kosyncError(c, http.StatusBadRequest, "invalid document")
	case err != nil:
		kosyncError(c, http.StatusInternalServerError, "progress read failed")
	default:
		c.JSON(http.StatusOK, progress)
	}
}
