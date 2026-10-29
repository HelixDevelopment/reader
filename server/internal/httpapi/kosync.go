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

func (s *Server) kosyncAvailable(c *gin.Context) bool {
	if s.KOSync == nil {
		Error(c, http.StatusServiceUnavailable, "KOReader sync is not configured")
		return false
	}
	return true
}

func (s *Server) kosyncAuthenticate(c *gin.Context) (string, bool) {
	username := c.GetHeader("x-auth-user")
	if err := s.KOSync.Authenticate(c.Request.Context(), username, c.GetHeader("x-auth-key")); err != nil {
		if errors.Is(err, kosync.ErrInvalidAuth) {
			Error(c, http.StatusUnauthorized, "invalid username or password")
		} else {
			Error(c, http.StatusInternalServerError, "authentication failed")
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
		Error(c, http.StatusBadRequest, "invalid user request")
		return
	}
	if err := s.KOSync.Register(c.Request.Context(), request.Username, request.Password); err != nil {
		switch {
		case errors.Is(err, kosync.ErrRegistrationDisabled):
			Error(c, http.StatusForbidden, "registration is disabled")
		case errors.Is(err, kosync.ErrInvalidInput):
			Error(c, http.StatusBadRequest, "invalid username or password")
		case errors.Is(err, kosync.ErrUserExists):
			Error(c, http.StatusConflict, "cannot create user")
		default:
			Error(c, http.StatusInternalServerError, "user creation failed")
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
		Error(c, http.StatusBadRequest, "invalid progress request")
		return
	}
	progress, err := s.KOSync.PutProgress(c.Request.Context(), username, progress)
	if errors.Is(err, kosync.ErrInvalidInput) {
		Error(c, http.StatusBadRequest, "missing or invalid progress fields")
		return
	}
	if err != nil {
		Error(c, http.StatusInternalServerError, "progress update failed")
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
		Error(c, http.StatusNotFound, "progress not found")
	case errors.Is(err, kosync.ErrInvalidInput):
		Error(c, http.StatusBadRequest, "invalid document")
	case err != nil:
		Error(c, http.StatusInternalServerError, "progress read failed")
	default:
		c.JSON(http.StatusOK, progress)
	}
}
